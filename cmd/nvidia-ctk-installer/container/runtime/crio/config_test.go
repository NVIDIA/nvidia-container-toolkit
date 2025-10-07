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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
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
		prepareEnvironment          func(*testing.T, *container.Options, *Options) error
		expectedSetupError          error
		assertSetupPostConditions   func(*testing.T, *container.Options, *Options) error
		expectedCleanupError        error
		assertCleanupPostConditions func(*testing.T, *container.Options, *Options) error
	}{
		{
			description: "config mode: top-level config does not exist",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/crio/crio.conf",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
			},
			options: Options{
				configMode: "config",
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, _ *Options) error {
				require.NoFileExists(t, co.TopLevelConfigPath)
				require.FileExists(t, co.DropInConfig)

				actual, err := os.ReadFile(co.DropInConfig)
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
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, _ *Options) error {
				require.NoFileExists(t, co.TopLevelConfigPath)
				require.NoFileExists(t, co.DropInConfig)
				return nil
			},
		},
		{
			description: "config mode: existing config without nvidia runtime",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/crio/crio.conf",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, _ *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

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
				err := os.WriteFile(co.TopLevelConfigPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, _ *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actualTopLevel, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expectedTopLevel := `[crio]
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

				require.Equal(t, expectedTopLevel, string(actualTopLevel))

				require.FileExists(t, co.DropInConfig)
				actual, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.runtime]

    [crio.runtime.runtimes]

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
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				require.NoFileExists(t, co.DropInConfig)

				actualTopLevel, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// Leaves original config unchanged
				expectedTopLevel := `[crio]
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
				require.Equal(t, expectedTopLevel, string(actualTopLevel))

				return nil
			},
		},
		{
			description: "config mode: existing config with nvidia runtime already present",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/crio/crio.conf",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, _ *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

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
				err := os.WriteFile(co.TopLevelConfigPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, _ *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actualTopLevel, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// TODO: Do we expect the top-level config to change? i.e. Should
				// we REMOVE the default_runtime = "nvidia" setting?
				expectedTopLevel := `[crio]
[crio.runtime]
default_runtime = "nvidia"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"

[crio.runtime.runtimes.nvidia]
runtime_path = "/old/path/nvidia-container-runtime"
runtime_type = "oci"
`

				require.Equal(t, expectedTopLevel, string(actualTopLevel))

				require.FileExists(t, co.DropInConfig)

				actual, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expected := `
[crio]

  [crio.runtime]
    default_runtime = "nvidia"

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
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actualTopLevel, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// TODO: Do we expect the top-level config to change? i.e. Should
				// we REMOVE the default_runtime = "nvidia" setting?
				expectedTopLevel := `[crio]
[crio.runtime]
default_runtime = "nvidia"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"

[crio.runtime.runtimes.nvidia]
runtime_path = "/old/path/nvidia-container-runtime"
runtime_type = "oci"
`

				require.Equal(t, expectedTopLevel, string(actualTopLevel))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "config mode: complex config with multiple settings",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/crio/crio.conf",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, _ *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

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
				err := os.WriteFile(co.TopLevelConfigPath, []byte(configContent), 0600)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, _ *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `[crio]
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
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `
[crio]

  [crio.runtime]

    [crio.runtime.runtimes]

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
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// Should restore to original complex config
				expected := `[crio]
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
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

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
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				hookPath := filepath.Join(o.hooksDir, o.hookFilename)
				require.FileExists(t, hookPath)

				actual, err := os.ReadFile(hookPath)
				require.NoError(t, err)

				expected := `{
  "version": "1.0.0",
  "hook": {
    "path": "/usr/bin/nvidia-container-runtime-hook",
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
`

				require.Equal(t, expected, string(actual))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				hookPath := filepath.Join(o.hooksDir, o.hookFilename)
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
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				hookPath := filepath.Join(o.hooksDir, o.hookFilename)
				require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0755))

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

				err := os.WriteFile(hookPath, []byte(existingHookJSON), 0600)
				require.NoError(t, err)

				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				hookPath := filepath.Join(o.hooksDir, o.hookFilename)
				require.FileExists(t, hookPath)

				actual, err := os.ReadFile(hookPath)
				require.NoError(t, err)

				expected := `{
  "version": "1.0.0",
  "hook": {
    "path": "/usr/bin/nvidia-container-runtime-hook",
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
`

				require.Equal(t, expected, string(actual))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				hookPath := filepath.Join(o.hooksDir, o.hookFilename)
				require.NoFileExists(t, hookPath)
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Update any paths as required
			testRoot := t.TempDir()
			tc.containerOptions.TopLevelConfigPath = strings.ReplaceAll(tc.containerOptions.TopLevelConfigPath, "{{ .testRoot }}", testRoot)
			tc.containerOptions.DropInConfig = strings.ReplaceAll(tc.containerOptions.DropInConfig, "{{ .testRoot }}", testRoot)
			tc.options.hooksDir = strings.ReplaceAll(tc.options.hooksDir, "{{ .testRoot }}", testRoot)
			tc.options.hookFilename = "99-nvidia.json"

			// Prepare the test environment
			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, &tc.containerOptions, &tc.options))
			}

			err := Setup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedSetupError, err)

			if tc.assertSetupPostConditions != nil {
				require.NoError(t, tc.assertSetupPostConditions(t, &tc.containerOptions, &tc.options))
			}

			err = Cleanup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedCleanupError, err)

			if tc.assertCleanupPostConditions != nil {
				require.NoError(t, tc.assertCleanupPostConditions(t, &tc.containerOptions, &tc.options))
			}
		})
	}
}
