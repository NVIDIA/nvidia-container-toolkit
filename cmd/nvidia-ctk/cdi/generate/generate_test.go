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

package generate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxa100"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGenerateSpec(t *testing.T) {
	// Create a temporary directory for config
	tmpDir, err := os.MkdirTemp("", "nvidia-container-toolkit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a temporary config file
	configContent := `
[nvidia-container-runtime]
mode = "nvml"
[[nvidia-container-runtime.modes.cdi]]
spec-dirs = ["/etc/cdi", "/usr/local/cdi"]
[nvidia-container-runtime.modes.csv]
mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"
	`
	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set XDG_CONFIG_HOME to point to our temporary directory
	oldXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDGConfigHome)

	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	driverRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-1")

	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description           string
		options               options
		expectedValidateError error
		expectedOptions       options
		expectedError         error
		expectedSpec          string
	}{
		{
			description: "default",
			options: options{
				format:     "yaml",
				mode:       "nvml",
				vendor:     "example.com",
				class:      "device",
				driverRoot: driverRoot,
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "nvml",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				driverRoot:        driverRoot,
				csv: struct {
					files          cli.StringSlice
					ignorePatterns cli.StringSlice
				}{
					files:          *cli.NewStringSlice("/etc/nvidia-container-runtime/host-files-for-container.d"),
					ignorePatterns: cli.StringSlice{},
				},
			},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: example.com/device
devices:
    - name: "0"
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia0
              hostPath: {{ .driverRoot }}/dev/nvidia0
    - name: all
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia0
              hostPath: {{ .driverRoot }}/dev/nvidia0
containerEdits:
    env:
        - NVIDIA_VISIBLE_DEVICES=void
    deviceNodes:
        - path: /dev/nvidiactl
          hostPath: {{ .driverRoot }}/dev/nvidiactl
    hooks:
        - hookName: createContainer
          path: /usr/bin/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - create-symlinks
            - --link
            - libcuda.so.1::/lib/x86_64-linux-gnu/libcuda.so
          env:
            - NVIDIA_CTK_DEBUG=false
        - hookName: createContainer
          path: /usr/bin/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - enable-cuda-compat
            - --host-driver-version=999.88.77
          env:
            - NVIDIA_CTK_DEBUG=false
        - hookName: createContainer
          path: /usr/bin/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - update-ldcache
            - --folder
            - /lib/x86_64-linux-gnu
          env:
            - NVIDIA_CTK_DEBUG=false
    mounts:
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/libcuda.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/libcuda.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c := command{
				logger: logger,
			}

			err := c.validateFlags(nil, &tc.options)
			require.ErrorIs(t, err, tc.expectedValidateError)
			// Set the ldconfig path to empty.
			// This is required during test because config.GetConfig() returns
			// the default ldconfig path, even if it is not set in the config file.
			tc.options.ldconfigPath = ""
			require.EqualValues(t, tc.expectedOptions, tc.options)

			// Set up a mock server, reusing the DGX A100 mock.
			server := dgxa100.New()
			// Override the driver version to match the version in our mock filesystem.
			server.SystemGetDriverVersionFunc = func() (string, nvml.Return) {
				return "999.88.77", nvml.SUCCESS
			}
			// Set the device count to 1 explicitly since we only have a single device node.
			server.DeviceGetCountFunc = func() (int, nvml.Return) {
				return 1, nvml.SUCCESS
			}
			for _, d := range server.Devices {
				// TODO: This is not implemented in the mock.
				(d.(*dgxa100.Device)).GetMaxMigDeviceCountFunc = func() (int, nvml.Return) {
					return 0, nvml.SUCCESS
				}
			}
			tc.options.nvmllib = server

			spec, err := c.generateSpec(&tc.options)
			require.ErrorIs(t, err, tc.expectedError)

			var buf bytes.Buffer
			_, err = spec.WriteTo(&buf)
			require.NoError(t, err)

			require.Equal(t, strings.ReplaceAll(tc.expectedSpec, "{{ .driverRoot }}", driverRoot), buf.String())
		})
	}
}
