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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

// containerdTestEnv defines the test environment for different containerd
// versions
type containerdTestEnv struct {
	name          string
	image         string
	configVersion int64
	pluginName    string
	// TODO: We could read this from the original config.
	cdiEnabledByDefault bool
}

// Define both containerd versions to test
var containerdEnvs = []*containerdTestEnv{
	{
		name:          "containerd-1.7",
		image:         "kindest/node:v1.30.0@sha256:047357ac0cfea04663786a612ba1eaba9702bef25227a794b52890dd8bcd692e",
		configVersion: 2,
		pluginName:    "io.containerd.grpc.v1.cri",
	},
	{
		name:          "containerd-2.1",
		image:         "docker.io/kindest/base:v20250521-31a79fd4",
		configVersion: 3,
		pluginName:    "io.containerd.cri.v1.runtime",
		// containerd >= 2.0 has CDI enabled by default
		cdiEnabledByDefault: true,
	},
}

type toolkitConfig struct {
	setAsDefault bool
	cdiEnabled   bool
}

var toolkitConfigVariants = []*toolkitConfig{
	{
		setAsDefault: true,
		cdiEnabled:   true,
	},
	{
		setAsDefault: false,
		cdiEnabled:   true,
	},
	{
		setAsDefault: false,
		cdiEnabled:   false,
	},
}

type testConfig struct {
	*containerdTestEnv
	*toolkitConfig
}

func (c *testConfig) name() string {
	return fmt.Sprintf("%s-default=%v-cdi=%v", c.containerdTestEnv.name, c.setAsDefault, c.cdiEnabled)
}

// Integration tests for containerd drop-in config functionality.
// These tests verify that nvidia-ctk runtime configure correctly applies
// configuration changes while preserving existing settings. The preservation
// validation is critical for containerd < 2.1 where plugin merge behavior
// (commit 598c632) requires duplicating the CRI plugin section.
var _ = Describe("containerd", Ordered, ContinueOnFailure, Label("container-runtime"), func() {
	var testsConfigs []*testConfig
	for _, env := range containerdEnvs {
		for _, variant := range toolkitConfigVariants {
			config := &testConfig{
				env,
				variant,
			}
			testsConfigs = append(testsConfigs, config)
		}
	}

	// Run all tests for each testConfig
	for _, tc := range testsConfigs {
		Context(tc.name(), Ordered, func() {
			var (
				nestedContainerRunner  Runner
				containerName          = "nvctk-e2e-containerd-tests"
				baselineConfig         *toml.Tree
				baselineConfigPlugins  *toml.Tree
				baselineConfigRuntimes map[string]any
			)

			// restartContainerdAndWait restarts containerd and waits for it
			// to be ready
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
				nestedContainerRunner, err = NewNestedContainerRunner(runner, tc.image, false, containerName, localCacheDir, false)
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

				// Restart containerd to ensure clean state before capturing baseline
				err = restartContainerdAndWait(nestedContainerRunner)
				Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd before baseline capture")

				// CAPTURE BASELINE: Get config BEFORE any modifications
				baselineOutput, _, err := nestedContainerRunner.Run("containerd config dump")
				Expect(err).ToNot(HaveOccurred(), "Failed to dump baseline configuration")

				baselineConfig, err = toml.Load(baselineOutput)
				Expect(err).ToNot(HaveOccurred(), "Failed to parse baseline configuration")

				// Get plugin configs for comparison
				baselineConfigPlugins, err = getPluginConfig(baselineConfig, tc.pluginName)
				Expect(err).ToNot(HaveOccurred())

				baselineConfigRuntimes, err = getRuntimesConfig(baselineConfigPlugins)
				Expect(err).ToNot(HaveOccurred())

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

			When("configuring containerd", func() {
				var (
					mergedConfig         *toml.Tree
					mergedConfigPlugins  *toml.Tree
					mergedConfigRuntimes map[string]any
				)

				BeforeAll(func(ctx context.Context) {
					// Apply nvidia-ctk configuration
					cmd := []string{"nvidia-ctk runtime configure --runtime=containerd"}

					if tc.setAsDefault {
						cmd = append(cmd, "--set-as-default")
					}
					if tc.cdiEnabled {
						cmd = append(cmd, "--cdi.enabled")
					}

					GinkgoLogr.Info("Applying nvidia-ctk configuration", "cmd", strings.Join(cmd, " "))
					_, _, err := nestedContainerRunner.Run(strings.Join(cmd, " "))
					Expect(err).ToNot(HaveOccurred(), "Failed to configure containerd")

					// Restart containerd to apply merged configuration
					err = restartContainerdAndWait(nestedContainerRunner)
					Expect(err).ToNot(HaveOccurred(), "Failed to restart containerd")

					// Get merged configuration
					mergedOutput, _, err := nestedContainerRunner.Run("containerd config dump")
					Expect(err).ToNot(HaveOccurred(), "Failed to dump merged configuration")

					mergedConfig, err = toml.Load(mergedOutput)
					Expect(err).ToNot(HaveOccurred(), "Failed to parse merged configuration")

					mergedConfigPlugins, err = getPluginConfig(mergedConfig, tc.pluginName)
					Expect(err).ToNot(HaveOccurred())

					mergedConfigRuntimes, err = getRuntimesConfig(mergedConfigPlugins)
					Expect(err).ToNot(HaveOccurred())
				})

				It("existing runtimes should remain unchanged", func(ctx context.Context) {
					for runtimeName, runtimeConfig := range baselineConfigRuntimes {
						Expect(mergedConfigRuntimes).To(HaveKeyWithValue(runtimeName, runtimeConfig))
					}
				})

				It("the nvidia runtime should be added with the correct options", func(ctx context.Context) {
					runcToml, err := toml.TreeFromMap(baselineConfigRuntimes["runc"].(map[string]any))
					Expect(err).ToNot(HaveOccurred())

					runcTomlString, err := runcToml.ToTomlString()
					Expect(err).ToNot(HaveOccurred())

					var addedRuntimes []string
					for runtimeName, runtimeConfig := range mergedConfigRuntimes {
						if _, ok := baselineConfigRuntimes[runtimeName]; ok {
							continue
						}
						addedRuntimes = append(addedRuntimes, runtimeName)

						Expect(runtimeConfig).To(
							HaveKeyWithValue(
								"options", HaveKeyWithValue("BinaryName", "/usr/bin/nvidia-container-runtime"),
							),
						)

						runtimeToml, err := toml.TreeFromMap(runtimeConfig.(map[string]any))
						Expect(err).ToNot(HaveOccurred())

						// TODO: for some reason the podsandboxer is set to "" in the updated config
						// but set to a valid string in the runc config. This is most likely due to
						// the config source that we're implicitly using.
						if runcSandboxer := runcToml.Get("sandboxer"); runcSandboxer != nil {
							runtimeToml.Set("sandboxer", runcSandboxer)
						}

						runcPath := runcToml.GetPath([]string{"options", "BinaryName"})
						if runcPath == nil {
							runtimeToml.DeletePath([]string{"options", "BinaryName"})
						}

						for _, option := range runcToml.Get("options").(*toml.Tree).Keys() {
							runcOption := runcToml.GetPath([]string{"options", option})
							switch v := runcOption.(type) {
							case int64:
								if runcOption.(int64) != 0 {
									GinkgoLogr.Info(fmt.Sprintf("non-zero option for %v: %+v %T", option, runcOption, v))
									continue
								}
							case string:
								if runcOption.(string) != "" {
									GinkgoLogr.Info(fmt.Sprintf("non-zero option for %v: %+v %T", option, runcOption, v))
									continue
								}
							case bool:
								if runcOption.(bool) {
									GinkgoLogr.Info(fmt.Sprintf("non-zero option for %v: %+v %T", option, runcOption, v))
									continue
								}
							default:
								panic(fmt.Sprintf("invalid type for option %v: %+v %T", option, runcOption, v))
							}
							runtimeToml.SetPath([]string{"options", option}, runcOption)
						}

						Expect(runtimeToml.ToTomlString()).To(BeEquivalentTo(runcTomlString))
					}

					Expect(addedRuntimes).To(Equal([]string{"nvidia"}))
				})

				It("should set the default runtime", func(ctx context.Context) {
					expectedDefault := "runc"
					if tc.setAsDefault {
						expectedDefault = "nvidia"
					}
					Expect(mergedConfigPlugins).To(
						WithTransform(
							func(t *toml.Tree) any {
								return t.GetPath([]string{"containerd", "default_runtime_name"})
							},
							BeEquivalentTo(expectedDefault),
						),
					)
				})

				It("should set cdi_enabled as expected", func(ctx context.Context) {
					Expect(mergedConfigPlugins).To(
						WithTransform(
							func(t *toml.Tree) any {
								return t.GetPath([]string{"enable_cdi"})
							},
							BeEquivalentTo(tc.cdiEnabledByDefault || tc.cdiEnabled),
						),
					)
				})

				It("should update the imports as expected", func(ctx context.Context) {
					Expect(mergedConfig).To(
						WithTransform(
							func(t *toml.Tree) any {
								return t.Get("imports")
							},
							ContainElement(HavePrefix("/etc/containerd/conf.d/")),
						),
					)
				})

				It("should preserve the other config options", func(ctx context.Context) {
					strippedMergedConfig, err := toml.TreeFromMap(mergedConfig.ToMap())
					Expect(err).ToNot(HaveOccurred())

					// We remove the nvidia runtime that was added.
					strippedMergedConfig.DeletePath([]string{"plugins", tc.pluginName, "containerd", "runtimes", "nvidia"})
					// We update the settings that we expect to change.
					for _, p := range [][]string{
						{"plugins", tc.pluginName, "containerd", "default_runtime_name"},
						{"plugins", tc.pluginName, "enable_cdi"},
						{"imports"},
					} {
						strippedMergedConfig.SetPath(p, baselineConfig.GetPath(p))
					}

					Expect(strippedMergedConfig.ToTomlString()).To(BeEquivalentTo(func() string {
						s, _ := baselineConfig.ToTomlString()
						return s
					}()))
				})
			})
		})
	}
})

// getPluginConfig navigates to the appropriate plugin configuration based on
// containerd version
func getPluginConfig(tree *toml.Tree, pluginName string) (*toml.Tree, error) {
	pluginTree := tree.GetPath([]string{"plugins", pluginName})
	if pluginTree == nil {
		return nil, fmt.Errorf("plugin %v not found", pluginName)
	}

	if pt, ok := pluginTree.(*toml.Tree); ok {
		return pt, nil
	}
	return nil, fmt.Errorf("plugin config is not a TOML tree")
}

// getRuntimesConfig gets the runtimes configuration from the plugin config
func getRuntimesConfig(pluginConfig *toml.Tree) (map[string]interface{}, error) {
	runtimes := pluginConfig.GetPath([]string{"containerd", "runtimes"})
	if runtimes == nil {
		return nil, fmt.Errorf("runtimes section not found")
	}

	// Handle both map and *toml.Tree types
	switch v := runtimes.(type) {
	case map[string]interface{}:
		return v, nil
	case *toml.Tree:
		return v.ToMap(), nil
	default:
		return nil, fmt.Errorf("runtimes is not a map or toml.Tree, got %T", runtimes)
	}
}
