/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package configure

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// TestConfigureLifecycle tests the complete configure command lifecycle for all runtimes
func TestConfigureLifecycle(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")

	logger := logger.Interface{Logger: testr.New(t)}

	testCases := []struct {
		description        string
		args               []string
		prepareEnvironment func(*testing.T, string) error
		expectedError      error
		assertConditions   func(*testing.T, string) error
	}{
		// Containerd v2 test cases
		{
			description: "containerd: config exists with imports",
			args: []string{
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				initialConfig := `version = 2
imports = ["/foo/bar/*.toml"]
`
				return os.WriteFile(configPath, []byte(initialConfig), 0600)
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				mainConfig := filepath.Join(testRoot, "etc/containerd/config.toml")
				content, err := os.ReadFile(mainConfig)
				require.NoError(t, err)

				expectedTemplate := `imports = ["/foo/bar/*.toml", "{{ .testRoot }}/etc/containerd/conf.d/*.toml"]
version = 2
`
				expected := strings.ReplaceAll(expectedTemplate, "{{ .testRoot }}", testRoot)

				require.Equal(t, expected, string(content))
				return nil
			},
		},
		{
			description: "containerd: v2 config does not exist with drop-in",
			args: []string{
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Verify main config was created with imports
				mainConfig := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, mainConfig)

				content, err := os.ReadFile(mainConfig)
				require.NoError(t, err)
				require.Contains(t, string(content), "imports")
				require.Contains(t, string(content), "version = 2")

				// Verify drop-in was created
				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				require.FileExists(t, dropIn)

				dropInContent, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(dropInContent), "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]")
				require.Contains(t, string(dropInContent), "BinaryName = \"/usr/bin/nvidia-container-runtime\"")

				return nil
			},
		},
		{
			description: "containerd: v2 existing config without nvidia runtime",
			args: []string{
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
				"--nvidia-set-as-default",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				initialConfig := `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				return os.WriteFile(configPath, []byte(initialConfig), 0600)
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Verify main config now has imports
				mainConfig := filepath.Join(testRoot, "etc/containerd/config.toml")
				content, err := os.ReadFile(mainConfig)
				require.NoError(t, err)
				require.Contains(t, string(content), "imports")

				// Verify drop-in was created with nvidia as default
				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				require.FileExists(t, dropIn)

				dropInContent, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(dropInContent), "default_runtime_name = \"nvidia\"")
				require.Contains(t, string(dropInContent), "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]")

				return nil
			},
		},
		// TODO: v1 config behavior is complex and differs between nvidia-ctk and installer
		// The installer modifies v1 configs directly, while nvidia-ctk appears to only
		// create drop-ins. This test is commented out until the expected behavior is clarified.
		/*
					{
						description: "containerd: v1 existing config",
						args: []string{
							"--runtime", "containerd",
							"--config", "{{ .testRoot }}/etc/containerd/config.toml",
							"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
						},
						prepareEnvironment: func(t *testing.T, testRoot string) error {
							configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
							require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

							// v1 config
							initialConfig := `version = 1

			[plugins]
			  [plugins.cri]
			    [plugins.cri.containerd]
			      default_runtime_name = "runc"

			      [plugins.cri.containerd.runtimes]
			        [plugins.cri.containerd.runtimes.runc]
			          runtime_type = "io.containerd.runtime.v1.linux"

			          [plugins.cri.containerd.runtimes.runc.options]
			            Runtime = "/usr/bin/runc"
			`
							return os.WriteFile(configPath, []byte(initialConfig), 0600)
						},
						assertConditions: func(t *testing.T, testRoot string) error {
							// The behavior for v1 configs needs clarification
							return nil
						},
					},
		*/
		{
			description: "containerd: enable CDI",
			args: []string{
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
				"--cdi.enabled",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				content, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(content), "enable_cdi = true")
				return nil
			},
		},
		{
			description: "containerd: custom runtime name and path",
			args: []string{
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
				"--nvidia-runtime-name", "gpu",
				"--nvidia-runtime-path", "/custom/path/nvidia-container-runtime",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				content, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(content), "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.gpu]")
				require.Contains(t, string(content), "BinaryName = \"/custom/path/nvidia-container-runtime\"")
				// Should NOT have legacy runtimes - only the single specified runtime is added
				require.NotContains(t, string(content), "gpu-cdi")
				require.NotContains(t, string(content), "gpu-legacy")
				return nil
			},
		},

		// CRI-O test cases
		{
			description: "crio: config does not exist with drop-in",
			args: []string{
				"--runtime", "crio",
				"--config", "{{ .testRoot }}/etc/crio/crio.conf",
				"--drop-in-config", "{{ .testRoot }}/etc/crio/conf.d/99-nvidia.toml",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Main config should not exist for crio when using drop-in
				mainConfig := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoFileExists(t, mainConfig)

				// Verify drop-in was created
				dropIn := filepath.Join(testRoot, "etc/crio/conf.d/99-nvidia.toml")
				require.FileExists(t, dropIn)

				content, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(content), "[crio.runtime.runtimes.nvidia]")
				require.Contains(t, string(content), "runtime_path = \"/usr/bin/nvidia-container-runtime\"")
				// Should NOT have legacy runtimes - only the single specified runtime is added
				require.NotContains(t, string(content), "nvidia-cdi")
				require.NotContains(t, string(content), "nvidia-legacy")

				return nil
			},
		},
		{
			description: "crio: existing config with nvidia runtime",
			args: []string{
				"--runtime", "crio",
				"--config", "{{ .testRoot }}/etc/crio/crio.conf",
				"--drop-in-config", "{{ .testRoot }}/etc/crio/conf.d/99-nvidia.toml",
				"--nvidia-set-as-default",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				configContent := `[crio]
[crio.runtime]
default_runtime = "runc"

[crio.runtime.runtimes.runc]
runtime_path = "/usr/bin/runc"
runtime_type = "oci"

[crio.runtime.runtimes.nvidia]
runtime_path = "/old/path/nvidia-container-runtime"
runtime_type = "oci"
`
				return os.WriteFile(configPath, []byte(configContent), 0600)
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Original config should remain unchanged
				mainConfig := filepath.Join(testRoot, "etc/crio/crio.conf")
				content, err := os.ReadFile(mainConfig)
				require.NoError(t, err)
				require.Contains(t, string(content), "/old/path/nvidia-container-runtime")

				// Drop-in should override settings
				dropIn := filepath.Join(testRoot, "etc/crio/conf.d/99-nvidia.toml")
				dropInContent, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(dropInContent), "default_runtime = \"nvidia\"")
				require.Contains(t, string(dropInContent), "runtime_path = \"/usr/bin/nvidia-container-runtime\"")

				return nil
			},
		},

		// Docker test cases
		{
			description: "docker: new JSON config",
			args: []string{
				"--runtime", "docker",
				"--config", "{{ .testRoot }}/etc/docker/daemon.json",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/docker/daemon.json")
				require.FileExists(t, configPath)

				content, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var dockerConfig map[string]interface{}
				err = json.Unmarshal(content, &dockerConfig)
				require.NoError(t, err)

				runtimes := dockerConfig["runtimes"].(map[string]interface{})
				require.Contains(t, runtimes, "nvidia")
				// Should NOT have legacy runtimes - only the single specified runtime is added
				require.NotContains(t, runtimes, "nvidia-cdi")
				require.NotContains(t, runtimes, "nvidia-legacy")

				return nil
			},
		},
		{
			description: "docker: existing config with set as default",
			args: []string{
				"--runtime", "docker",
				"--config", "{{ .testRoot }}/etc/docker/daemon.json",
				"--nvidia-set-as-default",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/docker/daemon.json")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				existingConfig := map[string]interface{}{
					"log-driver": "json-file",
					"log-opts": map[string]string{
						"max-size": "100m",
					},
					"storage-driver": "overlay2",
				}

				content, err := json.MarshalIndent(existingConfig, "", "    ")
				require.NoError(t, err)

				return os.WriteFile(configPath, content, 0600)
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/docker/daemon.json")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var dockerConfig map[string]interface{}
				err = json.Unmarshal(content, &dockerConfig)
				require.NoError(t, err)

				// Verify existing settings preserved
				require.Equal(t, "json-file", dockerConfig["log-driver"])
				require.Equal(t, "overlay2", dockerConfig["storage-driver"])

				// Verify nvidia runtime added and set as default
				require.Equal(t, "nvidia", dockerConfig["default-runtime"])
				runtimes := dockerConfig["runtimes"].(map[string]interface{})
				require.Contains(t, runtimes, "nvidia")

				return nil
			},
		},
		{
			description: "docker: enable CDI",
			args: []string{
				"--runtime", "docker",
				"--config", "{{ .testRoot }}/etc/docker/daemon.json",
				"--cdi.enabled",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/docker/daemon.json")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var dockerConfig map[string]interface{}
				err = json.Unmarshal(content, &dockerConfig)
				require.NoError(t, err)

				features := dockerConfig["features"].(map[string]interface{})
				require.Equal(t, true, features["cdi"])

				return nil
			},
		},

		// OCI Hook mode tests
		{
			description: "oci-hook: create hook file",
			args: []string{
				"--config-mode", "oci-hook",
				"--oci-hook-path", "{{ .testRoot }}/etc/containers/oci/hooks.d/99-nvidia.json",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/containers/oci/hooks.d/99-nvidia.json")
				require.FileExists(t, hookPath)

				content, err := os.ReadFile(hookPath)
				require.NoError(t, err)

				var hook map[string]interface{}
				err = json.Unmarshal(content, &hook)
				require.NoError(t, err)

				require.Equal(t, "1.0.0", hook["version"])
				require.Contains(t, hook["stages"], "prestart")

				hookSpec := hook["hook"].(map[string]interface{})
				require.Equal(t, defaultNVIDIARuntimeHookExpecutablePath, hookSpec["path"])

				when := hook["when"].(map[string]interface{})
				require.Equal(t, true, when["always"])

				return nil
			},
		},

		// Dry-run test
		{
			description: "dry-run: no files written",
			args: []string{
				"--dry-run",
				"--runtime", "containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Verify no files were created
				mainConfig := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoFileExists(t, mainConfig)

				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				require.NoFileExists(t, dropIn)

				return nil
			},
		},

		// Error cases
		{
			description:   "invalid runtime",
			args:          []string{"--runtime", "invalid"},
			expectedError: errUnrecognizedRuntime("invalid"),
		},
		{
			description: "invalid config-mode falls back to config-file",
			args: []string{
				"--config-mode", "invalid",
				"--runtime", "docker",
				"--config", "{{ .testRoot }}/etc/docker/daemon.json",
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Invalid mode should be converted to config-file mode
				configPath := filepath.Join(testRoot, "etc/docker/daemon.json")
				require.FileExists(t, configPath)

				content, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var dockerConfig map[string]interface{}
				err = json.Unmarshal(content, &dockerConfig)
				require.NoError(t, err)

				// Should have nvidia runtime added
				runtimes := dockerConfig["runtimes"].(map[string]interface{})
				require.Contains(t, runtimes, "nvidia")

				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create a temporary directory for the test
			testRoot := t.TempDir()

			// Replace placeholders in args
			args := make([]string, len(tc.args))
			for i, arg := range tc.args {
				args[i] = strings.ReplaceAll(arg, "{{ .testRoot }}", testRoot)
			}

			// Prepare the environment
			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, testRoot))
			}

			// Create and run the command
			cmd := NewCommand(logger)
			app := &cli.Command{
				Name:     "test",
				Commands: []*cli.Command{cmd},
			}

			// Prepend app name and command to args
			fullArgs := append([]string{"test", "configure"}, args...)

			err := app.Run(context.Background(), fullArgs)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			if tc.assertConditions != nil {
				require.NoError(t, tc.assertConditions(t, testRoot))
			}
		})
	}
}

// TestConfigureCommandLineSource tests using command source for config
func TestConfigureCommandLineSource(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")

	logger := logger.Interface{Logger: testr.New(t)}

	testCases := []struct {
		description        string
		runtime            string
		args               []string
		prepareEnvironment func(*testing.T, string) error
		expectedError      error
		assertConditions   func(*testing.T, string) error
	}{
		{
			description: "containerd: config from command",
			runtime:     "containerd",
			args: []string{
				"--runtime", "containerd",
				"--config-source", "command",
				"--executable-path", "{{ .testRoot }}/bin/containerd",
				"--config", "{{ .testRoot }}/etc/containerd/config.toml",
				"--drop-in-config", "{{ .testRoot }}/etc/containerd/conf.d/99-nvidia.toml",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				// Create a mock containerd executable that outputs config
				binPath := filepath.Join(testRoot, "bin/containerd")
				require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0755)) //nolint:gosec

				mockScript := `#!/bin/sh
if [ "$1" = "config" ] && [ "$2" = "dump" ]; then
cat <<EOF
version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
EOF
fi
`
				err := os.WriteFile(binPath, []byte(mockScript), 0755) //nolint:gosec
				require.NoError(t, err)

				// Create the config file path so the top-level config has a valid path to save to
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
				require.NoError(t, os.WriteFile(configPath, []byte("version = 2\n"), 0644)) //nolint:gosec

				// Also create the drop-in config directory
				dropInDir := filepath.Join(testRoot, "etc/containerd/config.d")
				require.NoError(t, os.MkdirAll(dropInDir, 0755)) //nolint:gosec

				return nil
			},
			assertConditions: func(t *testing.T, testRoot string) error {
				// Should create drop-in with nvidia runtime
				dropIn := filepath.Join(testRoot, "etc/containerd/conf.d/99-nvidia.toml")
				require.FileExists(t, dropIn)

				content, err := os.ReadFile(dropIn)
				require.NoError(t, err)
				require.Contains(t, string(content), "nvidia")

				// Check that the top-level config was updated with imports
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)

				configContent, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Contains(t, string(configContent), "imports")
				require.Contains(t, string(configContent), "*.toml")

				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			testRoot := t.TempDir()

			// Replace placeholders in args
			args := make([]string, len(tc.args))
			for i, arg := range tc.args {
				args[i] = strings.ReplaceAll(arg, "{{ .testRoot }}", testRoot)
			}

			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, testRoot))
			}

			cmd := NewCommand(logger)
			app := &cli.Command{
				Name:     "test",
				Commands: []*cli.Command{cmd},
			}

			fullArgs := append([]string{"test", "configure"}, args...)
			err := app.Run(context.Background(), fullArgs)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			if tc.assertConditions != nil {
				require.NoError(t, tc.assertConditions(t, testRoot))
			}
		})
	}
}

// Helper functions for expected errors
func errUnrecognizedRuntime(runtime string) error {
	return cli.Exit("unrecognized runtime '"+runtime+"'", 1)
}
