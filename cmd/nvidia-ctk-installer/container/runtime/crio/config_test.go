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
*/

package crio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

// TestCrioConfigLifecycle tests the complete Setup->Cleanup lifecycle for both config and hook modes
func TestCrioConfigLifecycle(t *testing.T) {
	c := &cli.Command{
		Name: "test",
	}

	testCases := []struct {
		description                 string
		containerOptions            container.Options
		options                     Options
		prepareEnvironment          func(*testing.T, string) error
		expectedSetupError          error
		assertSetupPostConditions   func(*testing.T, string) error
		expectedCleanupError        error
		assertCleanupPostConditions func(*testing.T, string) error
	}{
		{
			description: "config mode: top-level config does not exist",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.runtime]

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.nvidia]
        runtime_path = "/usr/bin/nvidia-container-runtime"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-cdi]
        runtime_path = "/usr/bin/nvidia-container-runtime.cdi"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-legacy]
        runtime_path = "/usr/bin/nvidia-container-runtime.legacy"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoFileExists(t, configPath)
				return nil
			},
		},
		{
			description: "config mode: existing config without nvidia runtime",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				configContent := `[crio]
[crio.runtime]
default_runtime = "crun"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"
runtime_root = "/run/crun"
monitor_path = "/usr/libexec/crio/conmon"

[crio.image]
signature_policy = "/etc/crio/policy.json"
`
				err := os.WriteFile(configPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.image]
    signature_policy = "/etc/crio/policy.json"

  [crio.runtime]
    default_runtime = "crun"

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/crun"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-cdi]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime.cdi"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-legacy]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime.legacy"
        runtime_root = "/run/crun"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				// Should restore to original config
				expected := `
[crio]

  [crio.image]
    signature_policy = "/etc/crio/policy.json"

  [crio.runtime]
    default_runtime = "crun"

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/crun"
        runtime_root = "/run/crun"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		{
			description: "config mode: existing config with nvidia runtime already present",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: true,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				configContent := `[crio]
[crio.runtime]
default_runtime = "nvidia"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"

[crio.runtime.runtimes.nvidia]
runtime_path = "/old/path/nvidia-container-runtime"
runtime_type = "oci"
`
				err := os.WriteFile(configPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.runtime]
    default_runtime = "nvidia"

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        runtime_path = "/usr/bin/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia]
        runtime_path = "/usr/bin/nvidia-container-runtime"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-cdi]
        runtime_path = "/usr/bin/nvidia-container-runtime.cdi"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-legacy]
        runtime_path = "/usr/bin/nvidia-container-runtime.legacy"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				// Note: cleanup removes nvidia runtimes but doesn't restore original default_runtime
				expected := `
[crio]

  [crio.runtime]

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        runtime_path = "/usr/bin/crun"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		{
			description: "config mode: complex config with multiple settings",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				configContent := `[crio]
[crio.runtime]
default_runtime = "crun"
conmon = "/usr/libexec/crio/conmon"
conmon_cgroup = "pod"
selinux = true

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"
runtime_root = "/run/crun"
monitor_path = "/usr/libexec/crio/conmon"

[crio.runtime.runtimes.runc]
runtime_path = "/usr/bin/runc"
runtime_type = "oci"
runtime_root = "/run/runc"

[crio.image]
signature_policy = "/etc/crio/policy.json"
insecure_registries = [
  "localhost:5000"
]

[crio.network]
network_dir = "/etc/cni/net.d/"
plugin_dirs = [
  "/opt/cni/bin",
  "/usr/libexec/cni"
]
`
				err := os.WriteFile(configPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.image]
    insecure_registries = ["localhost:5000"]
    signature_policy = "/etc/crio/policy.json"

  [crio.network]
    network_dir = "/etc/cni/net.d/"
    plugin_dirs = ["/opt/cni/bin", "/usr/libexec/cni"]

  [crio.runtime]
    conmon = "/usr/libexec/crio/conmon"
    conmon_cgroup = "pod"
    default_runtime = "crun"
    selinux = true

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/crun"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-cdi]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime.cdi"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.nvidia-legacy]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/nvidia-container-runtime.legacy"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.runc]
        runtime_path = "/usr/bin/runc"
        runtime_root = "/run/runc"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				actual, err := os.ReadFile(configPath)
				require.NoError(t, err)

				// Should restore to original complex config
				expected := `
[crio]

  [crio.image]
    insecure_registries = ["localhost:5000"]
    signature_policy = "/etc/crio/policy.json"

  [crio.network]
    network_dir = "/etc/cni/net.d/"
    plugin_dirs = ["/opt/cni/bin", "/usr/libexec/cni"]

  [crio.runtime]
    conmon = "/usr/libexec/crio/conmon"
    conmon_cgroup = "pod"
    default_runtime = "crun"
    selinux = true

    [crio.runtime.runtimes]

      [crio.runtime.runtimes.crun]
        monitor_path = "/usr/libexec/crio/conmon"
        runtime_path = "/usr/bin/crun"
        runtime_root = "/run/crun"
        runtime_type = "oci"

      [crio.runtime.runtimes.runc]
        runtime_path = "/usr/bin/runc"
        runtime_root = "/run/runc"
        runtime_type = "oci"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		// Hook mode test cases
		{
			description: "hook mode: no existing hook",
			containerOptions: container.Options{
				RuntimeName: "nvidia",
				RuntimeDir:  "/usr/bin",
				RestartMode: "none",
			},
			options: Options{
				configMode:   "hook",
				hooksDir:     "{{ .testRoot }}/etc/crio/hooks.d",
				hookFilename: "99-nvidia.json",
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.FileExists(t, hookPath)

				actual, err := os.ReadFile(hookPath)
				require.NoError(t, err)

				expectedContents := filepath.Join("/usr/bin", config.NVIDIAContainerRuntimeHookExecutable)
				expected := fmt.Sprintf(`{
  "version": "1.0.0",
  "hook": {
    "path": "%s",
    "args": [
      "nvidia-container-runtime-hook",
      "prestart"
    ],
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ]
  },
  "when": {
    "always": true,
    "commands": [
      ".*"
    ]
  },
  "stages": [
    "prestart"
  ]
}
`, expectedContents)

				require.Equal(t, expected, string(actual))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.NoFileExists(t, hookPath)
				return nil
			},
		},
		{
			description: "hook mode: existing hook file",
			containerOptions: container.Options{
				RuntimeName: "nvidia",
				RuntimeDir:  "/usr/bin",
				RestartMode: "none",
			},
			options: Options{
				configMode:   "hook",
				hooksDir:     "{{ .testRoot }}/etc/crio/hooks.d",
				hookFilename: "99-nvidia.json",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				hooksDir := filepath.Join(testRoot, "etc/crio/hooks.d")
				require.NoError(t, os.MkdirAll(hooksDir, 0755))

				// Create existing hook with old path
				existingHookJSON := `{
  "version": "1.0.0",
  "hook": {
    "path": "/old/path/nvidia-container-runtime-hook",
    "args": [
      "nvidia-container-runtime-hook",
      "prestart"
    ]
  },
  "when": {
    "always": true
  },
  "stages": [
    "prestart"
  ]
}`

				hookPath := filepath.Join(hooksDir, "99-nvidia.json")
				err := os.WriteFile(hookPath, []byte(existingHookJSON), 0600)
				require.NoError(t, err)

				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.FileExists(t, hookPath)

				actual, err := os.ReadFile(hookPath)
				require.NoError(t, err)

				expectedContents := filepath.Join("/usr/bin", config.NVIDIAContainerRuntimeHookExecutable)
				expected := fmt.Sprintf(`{
  "version": "1.0.0",
  "hook": {
    "path": "%s",
    "args": [
      "nvidia-container-runtime-hook",
      "prestart"
    ],
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ]
  },
  "when": {
    "always": true,
    "commands": [
      ".*"
    ]
  },
  "stages": [
    "prestart"
  ]
}
`, expectedContents)

				require.Equal(t, expected, string(actual))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.NoFileExists(t, hookPath)
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Update any paths as required
			testRoot := t.TempDir()
			tc.containerOptions.Config = strings.ReplaceAll(tc.containerOptions.Config, "{{ .testRoot }}", testRoot)
			tc.options.hooksDir = strings.ReplaceAll(tc.options.hooksDir, "{{ .testRoot }}", testRoot)

			// Prepare the test environment
			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, testRoot))
			}

			err := Setup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedSetupError, err)

			if tc.assertSetupPostConditions != nil {
				require.NoError(t, tc.assertSetupPostConditions(t, testRoot))
			}

			err = Cleanup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedCleanupError, err)

			if tc.assertCleanupPostConditions != nil {
				require.NoError(t, tc.assertCleanupPostConditions(t, testRoot))
			}
		})
	}
}
