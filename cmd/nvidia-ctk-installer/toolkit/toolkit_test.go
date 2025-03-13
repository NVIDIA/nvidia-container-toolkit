/**
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
**/

package toolkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestInstall(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")
	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	artifactRoot := filepath.Join(moduleRoot, "testdata", "installer", "artifacts")

	testCases := []struct {
		description     string
		hostRoot        string
		packageType     string
		cdiEnabled      bool
		expectedError   error
		expectedCdiSpec string
	}{
		{
			hostRoot:    "rootfs-empty",
			packageType: "deb",
		},
		{
			hostRoot:    "rootfs-empty",
			packageType: "rpm",
		},
		{
			hostRoot:      "rootfs-empty",
			packageType:   "deb",
			cdiEnabled:    true,
			expectedError: fmt.Errorf("no NVIDIA device nodes found"),
		},
		{
			hostRoot:    "rootfs-1",
			packageType: "deb",
			cdiEnabled:  true,
			expectedCdiSpec: `---
cdiVersion: 0.5.0
kind: example.com/class
devices:
    - name: all
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia0
              hostPath: /host/driver/root/dev/nvidia0
            - path: /dev/nvidiactl
              hostPath: /host/driver/root/dev/nvidiactl
            - path: /dev/nvidia-caps-imex-channels/channel0
              hostPath: /host/driver/root/dev/nvidia-caps-imex-channels/channel0
            - path: /dev/nvidia-caps-imex-channels/channel1
              hostPath: /host/driver/root/dev/nvidia-caps-imex-channels/channel1
            - path: /dev/nvidia-caps-imex-channels/channel2047
              hostPath: /host/driver/root/dev/nvidia-caps-imex-channels/channel2047
containerEdits:
    env:
        - NVIDIA_VISIBLE_DEVICES=void
    hooks:
        - hookName: createContainer
          path: {{ .toolkitRoot }}/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - create-symlinks
            - --link
            - libcuda.so.1::/lib/x86_64-linux-gnu/libcuda.so
        - hookName: createContainer
          path: {{ .toolkitRoot }}/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - update-ldcache
            - --folder
            - /lib/x86_64-linux-gnu
    mounts:
        - hostPath: /host/driver/root/lib/x86_64-linux-gnu/libcuda.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/libcuda.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - bind
`,
		},
	}

	for _, tc := range testCases {
		// hostRoot := filepath.Join(moduleRoot, "testdata", "lookup", tc.hostRoot)
		t.Run(tc.description, func(t *testing.T) {
			testRoot := t.TempDir()
			toolkitRoot := filepath.Join(testRoot, "toolkit-test")
			cdiOutputDir := filepath.Join(moduleRoot, "toolkit-test", "/var/cdi")
			sourceRoot := filepath.Join(artifactRoot, tc.packageType)
			options := Options{
				DriverRoot:        "/host/driver/root",
				DriverRootCtrPath: filepath.Join(moduleRoot, "testdata", "lookup", tc.hostRoot),
				CDI: cdiOptions{
					Enabled:   tc.cdiEnabled,
					outputDir: cdiOutputDir,
					kind:      "example.com/class",
				},
			}

			ti := NewInstaller(
				WithLogger(logger),
				WithToolkitRoot(toolkitRoot),
				WithSourceRoot(sourceRoot),
			)
			require.NoError(t, ti.ValidateOptions(&options))

			err := ti.Install(&cli.Context{}, &options)
			if tc.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.expectedError.Error())
			}

			require.DirExists(t, toolkitRoot)
			requireSymlink(t, toolkitRoot, "libnvidia-container.so.1", "libnvidia-container.so.99.88.77")
			requireSymlink(t, toolkitRoot, "libnvidia-container-go.so.1", "libnvidia-container-go.so.99.88.77")

			requireWrappedExecutable(t, toolkitRoot, "nvidia-cdi-hook")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-container-cli")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-container-runtime")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-container-runtime-hook")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-container-runtime.cdi")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-container-runtime.legacy")
			requireWrappedExecutable(t, toolkitRoot, "nvidia-ctk")

			requireSymlink(t, toolkitRoot, "nvidia-container-toolkit", "nvidia-container-runtime-hook")

			// TODO: Add checks for wrapper contents
			// grep -q -E "nvidia driver modules are not yet loaded, invoking runc directly" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime"
			// grep -q -E "exec runc \".@\"" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime"

			require.DirExists(t, filepath.Join(toolkitRoot, ".config"))
			require.DirExists(t, filepath.Join(toolkitRoot, ".config", "nvidia-container-runtime"))
			require.FileExists(t, filepath.Join(toolkitRoot, ".config", "nvidia-container-runtime", "config.toml"))

			cfgToml, err := config.New(config.WithConfigFile(filepath.Join(toolkitRoot, ".config", "nvidia-container-runtime", "config.toml")))
			require.NoError(t, err)

			cfg, err := cfgToml.Config()
			require.NoError(t, err)

			// Ensure that the config file has the required contents.
			// TODO: Add checks for additional config options.
			require.Equal(t, "/host/driver/root", cfg.NVIDIAContainerCLIConfig.Root)
			require.Equal(t, "@/host/driver/root/sbin/ldconfig", string(cfg.NVIDIAContainerCLIConfig.Ldconfig))
			require.EqualValues(t, filepath.Join(toolkitRoot, "nvidia-container-cli"), cfg.NVIDIAContainerCLIConfig.Path)
			require.EqualValues(t, filepath.Join(toolkitRoot, "nvidia-ctk"), cfg.NVIDIACTKConfig.Path)

			if len(tc.expectedCdiSpec) > 0 {
				cdiSpecFile := filepath.Join(cdiOutputDir, "example.com-class.yaml")
				require.FileExists(t, cdiSpecFile)
				info, err := os.Stat(cdiSpecFile)
				require.NoError(t, err)
				require.NotZero(t, info.Mode()&0004)
				contents, err := os.ReadFile(cdiSpecFile)
				require.NoError(t, err)
				require.Equal(t, strings.ReplaceAll(tc.expectedCdiSpec, "{{ .toolkitRoot }}", toolkitRoot), string(contents))
			}
		})
	}
}

func requireWrappedExecutable(t *testing.T, toolkitRoot string, expectedExecutable string) {
	requireExecutable(t, toolkitRoot, expectedExecutable)
	requireExecutable(t, toolkitRoot, expectedExecutable+".real")
}

func requireExecutable(t *testing.T, toolkitRoot string, expectedExecutable string) {
	executable := filepath.Join(toolkitRoot, expectedExecutable)
	require.FileExists(t, executable)
	info, err := os.Lstat(executable)
	require.NoError(t, err)
	require.Zero(t, info.Mode()&os.ModeSymlink)
	require.NotZero(t, info.Mode()&0111)
}

func requireSymlink(t *testing.T, toolkitRoot string, expectedLink string, expectedTarget string) {
	link := filepath.Join(toolkitRoot, expectedLink)
	require.FileExists(t, link)
	target, err := symlinks.Resolve(link)
	require.NoError(t, err)
	require.Equal(t, expectedTarget, target)
}
