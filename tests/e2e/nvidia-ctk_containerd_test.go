/*
 * Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/pelletier/go-toml"
)

const (
	// restartContainerdScript restarts containerd and waits for it to be ready
	restartContainerdScript = `
systemctl restart containerd
sleep 2
containerd --version
`
	// waitForContainerdScript waits for containerd to be ready
	waitForContainerdScript = `
for i in $(seq 1 10); do
	if ctr version > /dev/null 2>&1; then
		echo "containerd is ready"
		exit 0
	fi
	echo "Waiting for containerd to be ready..."
	sleep 1
done
echo "containerd failed to start"
exit 1
`
)

// containerdTestEnv defines the test environment for different containerd versions
type containerdTestEnv struct {
	name          string
	image         string
	configVersion int64
	pluginPath    string
}

// Define both containerd versions to test
var containerdEnvs = []containerdTestEnv{
	{
		name:          "containerd-1.7",
		image:         "kindest/node:v1.30.0@sha256:047357ac0cfea04663786a612ba1eaba9702bef25227a794b52890dd8bcd692e",
		configVersion: 2,
		pluginPath:    "io.containerd.grpc.v1.cri",
	},
	{
		name:          "containerd-2.1",
		image:         "docker.io/kindest/base:v20250521-31a79fd4",
		configVersion: 3,
		pluginPath:    "io.containerd.cri.v1.runtime",
	},
}

// Integration tests for containerd drop-in config functionality
var _ = Describe("containerd", Ordered, ContinueOnFailure, Label("container-runtime"), func() {
	// Run all tests for each containerd version
	for _, env := range containerdEnvs {
		Context(env.name, Ordered, func() {
			var (
				nestedContainerRunner          Runner
				containerName                  = "nvctk-e2e-containerd-tests-" + env.name
				originalTopLevelConfigContents string
				// originalTopLevelConfigToml     *toml.Tree
			)

			// restartContainerdAndWait restarts containerd and waits for it to be ready
			restartContainerdAndWait := func(runner Runner) error {
				_, _, err := runner.Run(restartContainerdScript)
				if err != nil {
					return fmt.Errorf("failed to restart containerd: %w", err)
				}

				_, _, err = runner.Run(waitForContainerdScript)
				if err != nil {
					return fmt.Errorf("containerd did not become ready after restart")
				}
				return nil
			}

			BeforeAll(func(ctx context.Context) {
				var err error

				// Create the nested container with the global cache mounted
				// TODO: This runner doesn't actually NEED GPU access.
				nestedContainerRunner, err = NewNestedContainerRunner(runner, env.image, false, containerName, localCacheDir)
				Expect(err).ToNot(HaveOccurred())

				// Store the contents of the original config.
				originalTopLevelConfigContents, _, err = nestedContainerRunner.Run("cat /etc/containerd/config.toml")
				Expect(err).ToNot(HaveOccurred())

				_, err = toml.Load(originalTopLevelConfigContents)
				Expect(err).ToNot(HaveOccurred())

				// Backup original containerd configuration
				_, _, err = nestedContainerRunner.Run(`
			# Backup the original conf.d directory
			if [ -d /etc/containerd/conf.d ]; then
				cp -r /etc/containerd/conf.d /tmp/containerd-conf.d.backup
			fi

			# Backup the original config.toml
			if [ -f /etc/containerd/config.toml ]; then
				cp /etc/containerd/config.toml /tmp/containerd-config.toml.backup
			fi
		`)
				Expect(err).ToNot(HaveOccurred(), "Failed to backup containerd configuration")

				// Ensure containerd is running
				_, _, err = nestedContainerRunner.Run(`
					# Start containerd if not running
					if ! systemctl is-active --quiet containerd; then
						systemctl start containerd
						sleep 2
					fi
				`)
				Expect(err).ToNot(HaveOccurred(), "Failed to ensure containerd is running")

				// Install the NVIDIA Container Toolkit packages
				_, _, err = toolkitInstaller.Install(nestedContainerRunner)
				Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")
			})

			AfterAll(func(ctx context.Context) {
				// Cleanup: remove the container
				_, _, err := runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", containerName))
				if err != nil {
					GinkgoLogr.Error(err, "failed to cleanup container", "container", containerName)
				}
			})

			BeforeEach(func(ctx context.Context) {
				// No setup needed - each test starts with the state from the previous test
			})

			AfterEach(func(ctx context.Context) {
				// Step 1: Restore containerd configuration from backup
				_, _, err := nestedContainerRunner.Run(`
			# Restore the original conf.d
			if [ -d /tmp/containerd-conf.d.backup ]; then
				rm -rf /etc/containerd/conf.d
				cp -r /tmp/containerd-conf.d.backup /etc/containerd/conf.d
			else
				# If no backup exists, just clean up
				rm -rf /etc/containerd/conf.d
				mkdir -p /etc/containerd/conf.d
			fi

			# Restore the original config.toml
			if [ -f /tmp/containerd-config.toml.backup ]; then
				cp /tmp/containerd-config.toml.backup /etc/containerd/config.toml
			fi
		`)
				Expect(err).ToNot(HaveOccurred(), "Failed to restore containerd configuration")

				// Step 2: Restart containerd and wait for it to be ready
				err = restartContainerdAndWait(nestedContainerRunner)
				Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd after configuration restore")
			})

			When("configuring containerd", func() {
				It("should add NVIDIA runtime using drop-in config without modifying the main config", func(ctx context.Context) {
					// Configure containerd using nvidia-ctk
					_, _, err := nestedContainerRunner.Run(`nvidia-ctk runtime configure --runtime=containerd --config=/etc/containerd/config.toml --drop-in-config=/etc/containerd/conf.d/99-nvidia.toml --set-as-default --cdi.enabled`)
					Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

					topLevelConfigContents, _, err := nestedContainerRunner.Run("cat /etc/containerd/config.toml")
					Expect(err).ToNot(HaveOccurred())

					_, err = toml.Load(topLevelConfigContents)
					Expect(err).ToNot(HaveOccurred())

					dropInConfigContents, _, err := nestedContainerRunner.Run("cat /etc/containerd/conf.d/99-nvidia.toml")
					Expect(err).ToNot(HaveOccurred())

					Expect(dropInConfigContents).ToNot(BeEmpty())

					// restart containerd
					err = restartContainerdAndWait(nestedContainerRunner)
					Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

					// Verify containerd loaded the config correctly
					output, _, err := nestedContainerRunner.Run(`containerd config dump`)
					Expect(err).ToNot(HaveOccurred())

					// Parse the TOML output
					config, err := toml.Load(output)
					Expect(err).ToNot(HaveOccurred(), "Failed to parse containerd config")

					BeAValidConfig := func(env *containerdTestEnv) types.GomegaMatcher {
						return And(
							WithTransform(
								func(c *toml.Tree) map[string]any {
									return c.ToMap()
								},
								And(
									HaveKeyWithValue("version", env.configVersion),
									HaveKeyWithValue("imports", WithTransform(func(is []any) []string {
										var basePaths []string
										for _, i := range is {
											basePaths = append(basePaths, filepath.Dir(i.(string)))
										}
										return basePaths
									},
										// TODO: We could do better at matching the following:
										// /etc/containerd/conf.d/99-nvidia.toml
										// /etc/containerd/conf.d/*.toml
										ContainElements("/etc/containerd/conf.d"),
									)),
								),
							),
							WithTransform(
								// Get the plugins config.
								func(c *toml.Tree) map[string]any {
									pt := c.GetPath([]string{"plugins", env.pluginPath})
									if pt != nil {
										return pt.(*toml.Tree).ToMap()
									}
									return nil
								},
								And(
									HaveKeyWithValue("enable_cdi", true),
									HaveKeyWithValue("containerd",
										And(
											// TODO: This should depend on whether we set the default.
											HaveKeyWithValue("default_runtime_name", "nvidia"),
											HaveKeyWithValue("runtimes", HaveKeyWithValue("nvidia",
												And(
													HaveKeyWithValue("runtime_type", "io.containerd.runc.v2"),
													HaveKeyWithValue("options",
														And(
															HaveKeyWithValue("BinaryName", "/usr/bin/nvidia-container-runtime"),
															HaveKeyWithValue("SystemdCgroup", true),
														),
													),
												),
											)),
										),
									),
								),
							),
						)
					}
					Expect(config).To(BeAValidConfig(&env))
				})
			})

			When("containerd has an existing kata runtime configured", func() {
				Context("when kata is the default runtime", func() {
					BeforeEach(func(ctx context.Context) {
						// Create containerd config with kata as default runtime
						configContent := ""
						if env.configVersion == 2 {
							// containerd 1.7.x
							configContent = `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "kata"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
          runtime_type = "io.containerd.kata.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata.options]
            ConfigPath = "/etc/kata-containers/configuration.toml"`
						} else {
							// containerd 2.x
							configContent = `version = 3

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "kata"

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.kata]
          runtime_type = "io.containerd.kata.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.kata.options]
            ConfigPath = "/etc/kata-containers/configuration.toml"`
						}

						_, _, err := nestedContainerRunner.Run(fmt.Sprintf(`
cat > /etc/containerd/config.toml <<'EOF'
%s
EOF
`, configContent))
						Expect(err).ToNot(HaveOccurred())

						// Restart containerd
						err = restartContainerdAndWait(nestedContainerRunner)
						Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")
					})

					It("should preserve kata as default when --set-as-default=false", func(ctx context.Context) {
						// Configure without setting as default
						_, _, err := nestedContainerRunner.Run(`nvidia-ctk runtime configure --runtime=containerd --config=/etc/containerd/config.toml --drop-in-config=/etc/containerd/conf.d/99-nvidia.toml --config-source=file --set-as-default=false --cdi.enabled`)
						Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

						// Restart containerd
						err = restartContainerdAndWait(nestedContainerRunner)
						Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

						// Verify configuration using helper
						expectedRuntimes := map[string]map[string]interface{}{
							"kata": {
								"runtime_type": "io.containerd.kata.v2",
								"ConfigPath":   "/etc/kata-containers/configuration.toml",
							},
							"nvidia": {
								"runtime_type": "",
								"BinaryName":   "/usr/bin/nvidia-container-runtime",
							},
						}

						verifyRuntimeConfiguration(nestedContainerRunner, env, "kata", expectedRuntimes)
					})

					It("should set nvidia as default when --set-as-default=true", func(ctx context.Context) {
						// Configure with nvidia as default
						_, _, err := nestedContainerRunner.Run(`nvidia-ctk runtime configure --runtime=containerd --config=/etc/containerd/config.toml --drop-in-config=/etc/containerd/conf.d/99-nvidia.toml --config-source=file --set-as-default --cdi.enabled`)
						Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

						// Restart containerd
						err = restartContainerdAndWait(nestedContainerRunner)
						Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

						// Verify configuration using helper
						expectedRuntimes := map[string]map[string]interface{}{
							"kata": {
								"runtime_type": "io.containerd.kata.v2",
								"ConfigPath":   "/etc/kata-containers/configuration.toml",
							},
							"nvidia": {
								"runtime_type": "",
								"BinaryName":   "/usr/bin/nvidia-container-runtime",
							},
						}

						verifyRuntimeConfiguration(nestedContainerRunner, env, "nvidia", expectedRuntimes)
					})
				})
			})

			When("using containerd with version 3 configuration format", func() {
				It("should correctly add NVIDIA runtime to v3 config structure", func(ctx context.Context) {
					// This test only applies to containerd versions that support v3 config
					if env.configVersion != 3 {
						Skip("This test only applies to containerd with config version 3")
					}
					// Create a v3 containerd config
					_, _, err := nestedContainerRunner.Run(`
cat > /etc/containerd/config.toml <<'EOF'
version = 3

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "runc"

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true
EOF
`)
					Expect(err).ToNot(HaveOccurred())

					// Configure containerd using nvidia-ctk
					_, _, err = nestedContainerRunner.Run(`nvidia-ctk runtime configure --runtime=containerd --config=/etc/containerd/config.toml --drop-in-config=/etc/containerd/conf.d/99-nvidia.toml --cdi.enabled`)
					Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

					// Verify the drop-in config uses v3 format
					output, _, err := nestedContainerRunner.Run(`cat /etc/containerd/conf.d/99-nvidia.toml`)
					Expect(err).ToNot(HaveOccurred())

					// Parse drop-in config to verify it's v3
					dropinConfig, err := parseContainerdConfig(output)
					Expect(err).ToNot(HaveOccurred(), "Failed to parse drop-in config")
					Expect(dropinConfig.Get("version")).To(Equal(int64(3)), "Drop-in config should be version 3")

					// Restart containerd
					err = restartContainerdAndWait(nestedContainerRunner)
					Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

					// Verify containerd loaded v3 config correctly
					output, _, err = nestedContainerRunner.Run(`containerd config dump`)
					Expect(err).ToNot(HaveOccurred())

					// Parse the TOML output
					config, err := parseContainerdConfig(output)
					Expect(err).ToNot(HaveOccurred(), "Failed to parse containerd config")

					// Verify version
					Expect(config.Get("version")).To(Equal(int64(3)), "Config should be version 3")

					// Get plugin configuration
					pluginConfig, err := getPluginConfig(config, env.configVersion)
					Expect(err).ToNot(HaveOccurred(), "Failed to get plugin config")

					// Get runtimes configuration
					runtimes, err := getRuntimesConfig(pluginConfig, env.configVersion)
					Expect(err).ToNot(HaveOccurred(), "Failed to get runtimes config")

					// Verify existing runc runtime is preserved with its settings
					runcRuntime, exists := runtimes["runc"]
					Expect(exists).To(BeTrue(), "runc runtime should exist")
					expectedRuncOptions := map[string]interface{}{
						"BinaryName":    "/usr/bin/runc",
						"SystemdCgroup": true,
					}
					err = validateRuntimeConfig(runcRuntime, "io.containerd.runc.v2", expectedRuncOptions)
					Expect(err).ToNot(HaveOccurred(), "runc runtime validation failed")

					// Verify NVIDIA runtime is properly added
					nvidiaRuntime, exists := runtimes["nvidia"]
					Expect(exists).To(BeTrue(), "nvidia runtime should exist")
					expectedNvidiaOptions := map[string]interface{}{
						"BinaryName":    "/usr/bin/nvidia-container-runtime",
						"SystemdCgroup": true,
					}
					err = validateRuntimeConfig(nvidiaRuntime, "", expectedNvidiaOptions)
					Expect(err).ToNot(HaveOccurred(), "NVIDIA runtime validation failed")

					// Verify CDI is enabled
					cdiEnabled, err := getCDIEnabled(pluginConfig, env.configVersion)
					Expect(err).ToNot(HaveOccurred(), "Failed to get CDI config")
					Expect(cdiEnabled).To(BeTrue(), "CDI should be enabled")
				})
			})

			When("containerd already uses import directives for modular configuration", func() {
				It("should preserve existing imports when adding NVIDIA drop-in config", func(ctx context.Context) {
					// Create a containerd config with existing imports
					customConfig := ""
					if env.configVersion == 2 {
						// containerd 1.7.x
						customConfig = `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".cni]
      conf_template = "/etc/cni/net.d/10-containerd.conf"`
					} else {
						// containerd 2.x
						customConfig = `version = 3

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    [plugins."io.containerd.cri.v1.images".registry]
      [plugins."io.containerd.cri.v1.images".registry.mirrors]
        [plugins."io.containerd.cri.v1.images".registry.mirrors."myregistry.io"]
          endpoint = ["https://myregistry.io"]`
					}

					mainConfig := fmt.Sprintf(`imports = ["/etc/containerd/conf.d/*.toml"]
version = %d`, env.configVersion)

					_, _, err := nestedContainerRunner.Run(fmt.Sprintf(`
# Create a custom config that will be imported
cat > /etc/containerd/conf.d/10-custom.toml <<'EOF'
%s
EOF

# Create main config with existing imports
cat > /etc/containerd/config.toml <<'EOF'
%s
EOF
`, customConfig, mainConfig))
					Expect(err).ToNot(HaveOccurred())

					// Configure containerd using nvidia-ctk
					_, _, err = nestedContainerRunner.Run(`nvidia-ctk runtime configure --runtime=containerd --config=/etc/containerd/config.toml --drop-in-config=/etc/containerd/conf.d/99-nvidia.toml`)
					Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

					// Restart containerd to load the new configuration
					err = restartContainerdAndWait(nestedContainerRunner)
					Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

					// Verify all configs are merged correctly using containerd config dump
					output, _, err := nestedContainerRunner.Run(`containerd config dump`)
					Expect(err).ToNot(HaveOccurred())

					// Parse the TOML output
					config, err := parseContainerdConfig(output)
					Expect(err).ToNot(HaveOccurred(), "Failed to parse containerd config")

					// Verify imports are preserved
					if env.configVersion == 2 {
						// For containerd 1.7, verify the imports are listed
						err = validateImports(config, []string{"/etc/containerd/conf.d/10-custom.toml", "/etc/containerd/conf.d/99-nvidia.toml"}, true)
						Expect(err).ToNot(HaveOccurred(), "Import validation failed")
					} else {
						// For containerd 2.x, verify registry mirror from custom import is preserved
						imagesPlugin := config.GetPath([]string{"plugins", "io.containerd.cri.v1.images"})
						if imagesPlugin != nil {
							if imagesTree, ok := imagesPlugin.(*toml.Tree); ok {
								registry := imagesTree.Get("registry")
								if registry != nil {
									if registryTree, ok := registry.(*toml.Tree); ok {
										mirrors := registryTree.Get("mirrors")
										Expect(mirrors).ToNot(BeNil(), "Registry mirrors should exist")
									}
								}
							}
						}
					}

					// Get plugin configuration
					pluginConfig, err := getPluginConfig(config, env.configVersion)
					Expect(err).ToNot(HaveOccurred(), "Failed to get plugin config")

					// Get runtimes configuration
					runtimes, err := getRuntimesConfig(pluginConfig, env.configVersion)
					Expect(err).ToNot(HaveOccurred(), "Failed to get runtimes config")

					// Verify NVIDIA runtime from drop-in is properly loaded
					nvidiaRuntime, exists := runtimes["nvidia"]
					Expect(exists).To(BeTrue(), "nvidia runtime should exist")
					expectedNvidiaOptions := map[string]interface{}{
						"BinaryName": "/usr/bin/nvidia-container-runtime",
					}
					err = validateRuntimeConfig(nvidiaRuntime, "", expectedNvidiaOptions)
					Expect(err).ToNot(HaveOccurred(), "NVIDIA runtime validation failed")
				})
			})
		}) // End Context for containerd version
	} // End for loop over containerd versions
})

// parseContainerdConfig parses the containerd config dump output into a TOML tree
func parseContainerdConfig(output string) (*toml.Tree, error) {
	return toml.Load(output)
}

// tomlTreeToMap converts a toml.Tree to a map[string]interface{} recursively
func tomlTreeToMap(tree *toml.Tree) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range tree.Keys() {
		value := tree.Get(key)
		switch v := value.(type) {
		case *toml.Tree:
			result[key] = tomlTreeToMap(v)
		default:
			result[key] = v
		}
	}
	return result
}

// getPluginConfig navigates to the appropriate plugin configuration based on containerd version
func getPluginConfig(tree *toml.Tree, version int64) (*toml.Tree, error) {
	var pluginPath []string
	if version == 2 {
		pluginPath = []string{"plugins", "io.containerd.grpc.v1.cri"}
	} else {
		pluginPath = []string{"plugins", "io.containerd.cri.v1.runtime"}
	}

	plugins := tree.Get("plugins")
	if plugins == nil {
		return nil, fmt.Errorf("plugins section not found")
	}

	pluginTree := tree.GetPath(pluginPath)
	if pluginTree == nil {
		return nil, fmt.Errorf("plugin path %v not found", pluginPath)
	}

	if pt, ok := pluginTree.(*toml.Tree); ok {
		return pt, nil
	}
	return nil, fmt.Errorf("plugin config is not a TOML tree")
}

// getRuntimesConfig gets the runtimes configuration from the plugin config
func getRuntimesConfig(pluginConfig *toml.Tree, _ int64) (map[string]interface{}, error) {
	containerdSection := pluginConfig.Get("containerd")
	if containerdSection == nil {
		return nil, fmt.Errorf("containerd section not found")
	}

	containerdTree, ok := containerdSection.(*toml.Tree)
	if !ok {
		return nil, fmt.Errorf("containerd section is not a TOML tree")
	}

	runtimes := containerdTree.Get("runtimes")
	if runtimes == nil {
		return nil, fmt.Errorf("runtimes section not found")
	}

	// Handle both map and *toml.Tree types
	switch v := runtimes.(type) {
	case map[string]interface{}:
		return v, nil
	case *toml.Tree:
		return tomlTreeToMap(v), nil
	default:
		return nil, fmt.Errorf("runtimes is not a map or toml.Tree, got %T", runtimes)
	}
}

// getCDIEnabled checks if CDI is enabled in the plugin configuration
func getCDIEnabled(pluginConfig *toml.Tree, _ int64) (bool, error) {
	cdiEnabled := pluginConfig.Get("enable_cdi")
	if cdiEnabled == nil {
		return false, nil // CDI not configured, default is false
	}

	if enabled, ok := cdiEnabled.(bool); ok {
		return enabled, nil
	}

	return false, fmt.Errorf("enable_cdi is not a boolean")
}

// getDefaultRuntime gets the default runtime name from the containerd configuration
func getDefaultRuntime(pluginConfig *toml.Tree) (string, error) {
	containerdSection := pluginConfig.Get("containerd")
	if containerdSection == nil {
		return "", fmt.Errorf("containerd section not found")
	}

	containerdTree, ok := containerdSection.(*toml.Tree)
	if !ok {
		return "", fmt.Errorf("containerd section is not a TOML tree")
	}

	defaultRuntime := containerdTree.Get("default_runtime_name")
	if defaultRuntime == nil {
		return "", nil // No default runtime set
	}

	if runtime, ok := defaultRuntime.(string); ok {
		return runtime, nil
	}

	return "", fmt.Errorf("default_runtime_name is not a string")
}

// validateRuntimeConfig validates a specific runtime configuration
func validateRuntimeConfig(runtime interface{}, expectedType string, expectedOptions map[string]interface{}) error {
	var runtimeMap map[string]interface{}

	// Handle both map and *toml.Tree types
	switch v := runtime.(type) {
	case map[string]interface{}:
		runtimeMap = v
	case *toml.Tree:
		runtimeMap = tomlTreeToMap(v)
	default:
		return fmt.Errorf("runtime is not a map or toml.Tree, got %T", runtime)
	}

	// Check runtime type
	runtimeType, ok := runtimeMap["runtime_type"].(string)
	if !ok {
		return fmt.Errorf("runtime_type not found or not a string")
	}

	if expectedType != "" {
		// For empty expectedType, we don't check the runtime type
		// For nvidia runtime, we accept various runtime types
		if expectedType == "nvidia" {
			validTypes := []string{"io.containerd.runc.v2", "io.containerd.nvidia.v1", "io.containerd.nvidia.v2"}
			found := false
			for _, validType := range validTypes {
				if runtimeType == validType {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected nvidia-compatible runtime_type, got %s", runtimeType)
			}
		} else if runtimeType != expectedType {
			return fmt.Errorf("expected runtime_type %s, got %s", expectedType, runtimeType)
		}
	}

	// Check options if provided
	if len(expectedOptions) > 0 {
		var options map[string]interface{}

		// Handle both map and *toml.Tree types for options
		switch v := runtimeMap["options"].(type) {
		case map[string]interface{}:
			options = v
		case *toml.Tree:
			options = tomlTreeToMap(v)
		case nil:
			return fmt.Errorf("options not found")
		default:
			return fmt.Errorf("options is not a map or toml.Tree, got %T", v)
		}

		for key, expectedValue := range expectedOptions {
			actualValue, exists := options[key]
			if !exists {
				return fmt.Errorf("option %s not found", key)
			}

			// Handle different types of comparisons
			switch expected := expectedValue.(type) {
			case string:
				actual, ok := actualValue.(string)
				if !ok || actual != expected {
					return fmt.Errorf("option %s: expected %v, got %v", key, expected, actualValue)
				}
			case bool:
				actual, ok := actualValue.(bool)
				if !ok || actual != expected {
					return fmt.Errorf("option %s: expected %v, got %v", key, expected, actualValue)
				}
			default:
				if actualValue != expectedValue {
					return fmt.Errorf("option %s: expected %v, got %v", key, expected, actualValue)
				}
			}
		}
	}

	return nil
}

// validateImports validates the imports configuration
func validateImports(tree *toml.Tree, expectedImports []string, partialMatch bool) error {
	imports := tree.Get("imports")
	if imports == nil {
		if len(expectedImports) == 0 {
			return nil
		}
		return fmt.Errorf("imports not found")
	}

	importsList, ok := imports.([]interface{})
	if !ok {
		return fmt.Errorf("imports is not a list")
	}

	importStrings := make([]string, 0, len(importsList))
	for _, imp := range importsList {
		if impStr, ok := imp.(string); ok {
			importStrings = append(importStrings, impStr)
		}
	}

	if partialMatch {
		// Check that all expected imports are present (but there may be more)
		for _, expected := range expectedImports {
			found := false
			for _, actual := range importStrings {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected import %s not found in %v", expected, importStrings)
			}
		}
	} else {
		// Exact match
		if len(importStrings) != len(expectedImports) {
			return fmt.Errorf("expected %d imports, got %d", len(expectedImports), len(importStrings))
		}
		for i, expected := range expectedImports {
			if i >= len(importStrings) || importStrings[i] != expected {
				return fmt.Errorf("import mismatch at index %d: expected %s, got %v", i, expected, importStrings)
			}
		}
	}

	return nil
}

// verifyRuntimeConfiguration verifies the entire runtime configuration including default runtime, runtimes, and CDI
func verifyRuntimeConfiguration(runner Runner, env containerdTestEnv, expectedDefault string, expectedRuntimes map[string]map[string]interface{}) {
	output, _, err := runner.Run(`containerd config dump`)
	Expect(err).ToNot(HaveOccurred())

	config, err := parseContainerdConfig(output)
	Expect(err).ToNot(HaveOccurred())

	pluginConfig, err := getPluginConfig(config, env.configVersion)
	Expect(err).ToNot(HaveOccurred())

	// Verify default runtime
	defaultRuntime, err := getDefaultRuntime(pluginConfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(defaultRuntime).To(Equal(expectedDefault))

	// Verify all expected runtimes
	runtimes, err := getRuntimesConfig(pluginConfig, env.configVersion)
	Expect(err).ToNot(HaveOccurred())

	for name, expectedOptions := range expectedRuntimes {
		runtime, exists := runtimes[name]
		Expect(exists).To(BeTrue(), fmt.Sprintf("%s runtime should exist", name))

		runtimeType := expectedOptions["runtime_type"].(string)
		delete(expectedOptions, "runtime_type")

		err = validateRuntimeConfig(runtime, runtimeType, expectedOptions)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s runtime validation failed", name))
	}

	// Verify CDI is enabled
	cdiEnabled, err := getCDIEnabled(pluginConfig, env.configVersion)
	Expect(err).ToNot(HaveOccurred())
	Expect(cdiEnabled).To(BeTrue())
}
