/*
# Copyright 2024 NVIDIA CORPORATION
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

package containerd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
)

// TestContainerdConfigLifecycle tests the complete Setup->Cleanup lifecycle for both v1 and v2 configs.
func TestContainerdConfigLifecycle(t *testing.T) {
	c := &cli.Command{
		Name: "test",
	}

	// TODO: Add test case for v1 and DropInConfig. This should fail.
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
		// V1 test cases
		// Note: We don't test "v1: top-level config does not exist" because new configs
		// are always created as v2 configs by the containerd package
		{
			description: "v1: existing config without nvidia runtime",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				EnableCDI:          true,
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial v1 config
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
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "runc"
      enable_cdi = true

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            Runtime = "/usr/bin/nvidia-container-runtime"

        [plugins.cri.containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            Runtime = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins.cri.containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            Runtime = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "runc"
      enable_cdi = true

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		{
			description: "v1: existing config with default_runtime_name and OPTIONS inheritance",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial v1 config without a default runtime set
				// This tests OPTIONS inheritance from an existing runtime
				initialConfig := `version = 1

[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
            Root = "/run/containerd/runc"
            ShimDebug = true
            SystemdCgroup = true
            NoPivotRoot = false
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "nvidia"

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            Runtime = "/usr/bin/nvidia-container-runtime"
            ShimDebug = true
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            Runtime = "/usr/bin/nvidia-container-runtime.cdi"
            ShimDebug = true
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            Runtime = "/usr/bin/nvidia-container-runtime.legacy"
            ShimDebug = true
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            Runtime = "/usr/bin/runc"
            ShimDebug = true
            SystemdCgroup = true
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            Runtime = "/usr/bin/runc"
            ShimDebug = true
            SystemdCgroup = true
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		{
			description: "v1: OPTIONS inheritance from default runtime specified by default_runtime_name",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false, // Don't change the default runtime
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Create config with default_runtime_name pointing to custom runtime
				// This tests that OPTIONS are inherited from the default runtime
				initialConfig := `version = 1

[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      default_runtime_name = "custom"

      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.custom]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.custom.options]
            Runtime = "/usr/bin/custom-runtime"
            Root = "/custom/root"
            ShimDebug = false
            SystemdCgroup = true
            NoPivotRoot = true
            CustomOption = "custom-value"

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// Verify that nvidia runtimes inherit OPTIONS from the default runtime (custom)
				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "custom"

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.custom]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.custom.options]
            CustomOption = "custom-value"
            NoPivotRoot = true
            Root = "/custom/root"
            Runtime = "/usr/bin/custom-runtime"
            ShimDebug = false
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            CustomOption = "custom-value"
            NoPivotRoot = true
            Root = "/custom/root"
            Runtime = "/usr/bin/nvidia-container-runtime"
            ShimDebug = false
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            CustomOption = "custom-value"
            NoPivotRoot = true
            Root = "/custom/root"
            Runtime = "/usr/bin/nvidia-container-runtime.cdi"
            ShimDebug = false
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            CustomOption = "custom-value"
            NoPivotRoot = true
            Root = "/custom/root"
            Runtime = "/usr/bin/nvidia-container-runtime.legacy"
            ShimDebug = false
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// After cleanup, should return to original state
				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "custom"

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.custom]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.custom.options]
            CustomOption = "custom-value"
            NoPivotRoot = true
            Root = "/custom/root"
            Runtime = "/usr/bin/custom-runtime"
            ShimDebug = false
            SystemdCgroup = true

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		{
			description: "v1: existing config with default_runtime_name set and restoration on cleanup",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial v1 config with default_runtime_name set to "runc"
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
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]
      default_runtime_name = "nvidia"

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            Runtime = "/usr/bin/nvidia-container-runtime"

        [plugins.cri.containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            Runtime = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins.cri.containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            Runtime = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// Current implementation limitation: default_runtime_name is deleted entirely
				// when the default runtime is removed, rather than being restored
				expected := `version = 1

[plugins]

  [plugins.cri]

    [plugins.cri.containerd]

      [plugins.cri.containerd.runtimes]

        [plugins.cri.containerd.runtimes.runc]
          runtime_type = "io.containerd.runtime.v1.linux"

          [plugins.cri.containerd.runtimes.runc.options]
            Runtime = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))
				return nil
			},
		},
		// V2 test cases
		{
			description: "v2: top-level config does not exist",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2
`
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoFileExists(t, co.TopLevelConfigPath)
				require.NoFileExists(t, co.DropInConfig)
				return nil
			},
		},
		{
			description: "v2: top-level config does not exist with drop-in-config-host-path",
			containerOptions: container.Options{
				TopLevelConfigPath:   "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:         "{{ .testRoot }}/conf.d/99-nvidia.toml",
				DropInConfigHostPath: "/some/host/path/conf.d/99-nvidia.toml",
				RuntimeName:          "nvidia",
				RuntimeDir:           "/usr/bin",
				SetAsDefault:         false,
				RestartMode:          "none",
				ExecutablePath:       "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["/some/host/path/conf.d/*.toml"]
version = 2
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfigHostPath)
				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoFileExists(t, co.TopLevelConfigPath)
				require.NoFileExists(t, co.DropInConfig)
				return nil
			},
		},
		{
			description: "v2: existing config without nvidia runtime and CDI enabled",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				EnableCDI:          true,
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial config
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
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

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

				// TODO (follow-up): Add a function to compare toml files by contents.
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]
    enable_cdi = true

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// TODO: What is the expectation here? Do we expect that the imports \
				// are updated.
				// TODO: Add a test case, where the original config consists of only imports and version
				// with the imports referring to another location.
				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

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
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "v2: existing config with nvidia runtime using old path already present",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				initialConfig := `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/old/path/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/old/path/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`

				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				// If file exists, verify nvidia runtimes were removed
				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// TODO: Do we expect that the default_runtime = nvidia be removed?
				// TODO: Should the `nvidia` runtimes be removed from the top-level config?
				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`

				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "v2: complex config with multiple plugins and settings",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				EnableCDI:          true,
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
				ContainerRuntimeModesCDIAnnotationPrefixes: []string{"cdi.k8s.io"},
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial complex config
				initialConfig := `version = 2
root = "/var/lib/containerd"
state = "/run/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom]
          runtime_type = "io.containerd.custom.v1"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom.options]
            TypeUrl = "custom.runtime/options"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
root = "/var/lib/containerd"
state = "/run/containerd"
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom]
          runtime_type = "io.containerd.custom.v1"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom.options]
            TypeUrl = "custom.runtime/options"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"
`
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]
    enable_cdi = true

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom]
          runtime_type = "io.containerd.custom.v1"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom.options]
            TypeUrl = "custom.runtime/options"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          container_annotations = ["cdi.k8s.io*"]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            SystemdCgroup = true

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          container_annotations = ["cdi.k8s.io*"]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            SystemdCgroup = true

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          container_annotations = ["cdi.k8s.io*"]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            SystemdCgroup = true

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
root = "/var/lib/containerd"
state = "/run/containerd"
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom]
          runtime_type = "io.containerd.custom.v1"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.custom.options]
            TypeUrl = "custom.runtime/options"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "v2: existing config without default runtime (SetAsDefault=false)",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial config without any default runtime
				initialConfig := `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`

				require.Equal(t, expectedDropIn, string(actualDropIn))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "v2: existing config without default runtime (SetAsDefault=true)",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial config without any default runtime
				initialConfig := `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`

				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// TODO: Add test where imports are already specified.
				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		// V3 test cases
		{
			description: "v3: minimal config without nvidia runtime",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       false,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write v3 config with runc runtime
				initialConfig := `version = 3

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 3

[plugins]

  [plugins."io.containerd.cri.v1.runtime"]

    [plugins."io.containerd.cri.v1.runtime".containerd]

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 3

[plugins]

  [plugins."io.containerd.cri.v1.runtime"]

    [plugins."io.containerd.cri.v1.runtime".containerd]

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expectedDropIn, string(actualDropIn))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// Should return to original v3 config with just runc
				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
version = 3

[plugins]

  [plugins."io.containerd.cri.v1.runtime"]

    [plugins."io.containerd.cri.v1.runtime".containerd]

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
		{
			description: "v3: existing config with runtime and OPTIONS inheritance",
			containerOptions: container.Options{
				TopLevelConfigPath: "{{ .testRoot }}/etc/containerd/config.toml",
				DropInConfig:       "{{ .testRoot }}/conf.d/99-nvidia.toml",
				RuntimeName:        "nvidia",
				RuntimeDir:         "/usr/bin",
				SetAsDefault:       true,
				RestartMode:        "none",
				ExecutablePath:     "not-containerd",
			},
			options: Options{
				runtimeType: "io.containerd.runc.v2",
			},
			prepareEnvironment: func(t *testing.T, co *container.Options, o *Options) error {
				require.NoError(t, os.MkdirAll(filepath.Dir(co.TopLevelConfigPath), 0755))

				// Write initial v3 config with a runtime that has custom OPTIONS
				initialConfig := `version = 3

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
            NoPivotRoot = false
            Root = "/run/containerd/runc"
`
				require.NoError(t, os.WriteFile(co.TopLevelConfigPath, []byte(initialConfig), 0600))
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
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
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true
`
				require.Equal(t, expected, string(actual))

				require.FileExists(t, co.DropInConfig)

				actualDropIn, err := os.ReadFile(co.DropInConfig)
				require.NoError(t, err)

				expectedDropIn := `version = 3

[plugins]

  [plugins."io.containerd.cri.v1.runtime"]

    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-cdi]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.cdi"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-legacy]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "/usr/bin/nvidia-container-runtime.legacy"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true
`
				require.Equal(t, expectedDropIn, string(actualDropIn))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, co *container.Options, o *Options) error {
				require.FileExists(t, co.TopLevelConfigPath)

				actual, err := os.ReadFile(co.TopLevelConfigPath)
				require.NoError(t, err)

				// After cleanup, should return to original state
				expected := `imports = ["` + filepath.Dir(co.DropInConfig) + `/*.toml"]
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
            NoPivotRoot = false
            Root = "/run/containerd/runc"
            SystemdCgroup = true
`
				require.Equal(t, expected, string(actual))

				require.NoFileExists(t, co.DropInConfig)

				return nil
			},
		},
	}

	for _, tc := range testCases {
		// Set default options that would normally be set by the CLI.
		if tc.containerOptions.ConfigSources == nil {
			tc.containerOptions.ConfigSources = []string{"command", "file"}
		}
		t.Run(tc.description, func(t *testing.T) {
			// Create a temporary directory for the test
			testRoot := t.TempDir()

			// Update paths
			tc.containerOptions.TopLevelConfigPath = strings.ReplaceAll(tc.containerOptions.TopLevelConfigPath, "{{ .testRoot }}", testRoot)
			tc.containerOptions.DropInConfig = strings.ReplaceAll(tc.containerOptions.DropInConfig, "{{ .testRoot }}", testRoot)
			tc.containerOptions.RuntimeDir = strings.ReplaceAll(tc.containerOptions.RuntimeDir, "{{ .testRoot }}", testRoot)
			var testConfigSources []string
			for _, configSource := range tc.containerOptions.ConfigSources {
				testConfigSources = append(testConfigSources, strings.ReplaceAll(configSource, "{{ .testRoot }}", testRoot))
			}
			tc.containerOptions.ConfigSources = testConfigSources

			// Prepare the environment
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
