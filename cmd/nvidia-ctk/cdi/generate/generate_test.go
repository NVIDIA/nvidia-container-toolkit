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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxa100"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGenerateSpec(t *testing.T) {
	defer devices.SetAllForTest()()

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
			description: "invalid device id",
			options: options{
				format:     "yaml",
				mode:       "nvml",
				vendor:     "example.com",
				class:      "device",
				deviceIDs:  []string{"99"},
				driverRoot: driverRoot,
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "nvml",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				deviceIDs:         []string{"99"},
				driverRoot:        driverRoot,
			},
			expectedError: fmt.Errorf("failed to create device CDI specs: failed to construct device spec generators: failed to get device handle from index: ERROR_INVALID_ARGUMENT"),
		},
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
        - NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu
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
            - --folder
            - /lib/x86_64-linux-gnu/vdpau
          env:
            - NVIDIA_CTK_DEBUG=false
        - hookName: createContainer
          path: /usr/bin/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - disable-device-node-modification
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
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
`,
		},
		{
			description: "disableHooks1",
			options: options{
				format:        "yaml",
				mode:          "nvml",
				vendor:        "example.com",
				class:         "device",
				driverRoot:    driverRoot,
				disabledHooks: []string{"enable-cuda-compat"},
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "nvml",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				driverRoot:        driverRoot,
				disabledHooks:     []string{"enable-cuda-compat"},
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
        - NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu
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
            - update-ldcache
            - --folder
            - /lib/x86_64-linux-gnu
            - --folder
            - /lib/x86_64-linux-gnu/vdpau
          env:
            - NVIDIA_CTK_DEBUG=false
        - hookName: createContainer
          path: /usr/bin/nvidia-cdi-hook
          args:
            - nvidia-cdi-hook
            - disable-device-node-modification
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
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
`,
		},
		{
			description: "disableHooks2",
			options: options{
				format:        "yaml",
				mode:          "nvml",
				vendor:        "example.com",
				class:         "device",
				driverRoot:    driverRoot,
				disabledHooks: []string{"enable-cuda-compat", "update-ldcache"},
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "nvml",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				driverRoot:        driverRoot,
				disabledHooks:     []string{"enable-cuda-compat", "update-ldcache"},
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
        - NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu
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
            - disable-device-node-modification
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
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
`,
		},
		{
			description: "disableHooksAll",
			options: options{
				format:        "yaml",
				mode:          "nvml",
				vendor:        "example.com",
				class:         "device",
				driverRoot:    driverRoot,
				disabledHooks: []string{"all"},
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "nvml",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				driverRoot:        driverRoot,
				disabledHooks:     []string{"all"},
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
        - NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu
        - NVIDIA_VISIBLE_DEVICES=void
    deviceNodes:
        - path: /dev/nvidiactl
          hostPath: {{ .driverRoot }}/dev/nvidiactl
    mounts:
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/libcuda.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/libcuda.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
`,
		},
		{
			description: "enableChmodHook",
			options: options{
				format:        "yaml",
				mode:          "management",
				vendor:        "example.com",
				class:         "device",
				driverRoot:    driverRoot,
				enabledHooks:  []string{"chmod"},
				disabledHooks: []string{"enable-cuda-compat", "update-ldcache", "disable-device-node-modification"},
			},
			expectedOptions: options{
				format:            "yaml",
				mode:              "management",
				vendor:            "example.com",
				class:             "device",
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
				driverRoot:        driverRoot,
				enabledHooks:      []string{"chmod"},
				disabledHooks:     []string{"enable-cuda-compat", "update-ldcache", "disable-device-node-modification"},
			},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: example.com/device
devices:
    - name: all
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia0
              hostPath: {{ .driverRoot }}/dev/nvidia0
            - path: /dev/nvidiactl
              hostPath: {{ .driverRoot }}/dev/nvidiactl
            - path: /dev/nvidia-caps-imex-channels/channel0
              hostPath: {{ .driverRoot }}/dev/nvidia-caps-imex-channels/channel0
            - path: /dev/nvidia-caps-imex-channels/channel1
              hostPath: {{ .driverRoot }}/dev/nvidia-caps-imex-channels/channel1
            - path: /dev/nvidia-caps-imex-channels/channel2047
              hostPath: {{ .driverRoot }}/dev/nvidia-caps-imex-channels/channel2047
            - path: /dev/nvidia-caps/nvidia-cap1
              hostPath: {{ .driverRoot }}/dev/nvidia-caps/nvidia-cap1
        hooks:
            - hookName: createContainer
              path: /usr/bin/nvidia-cdi-hook
              args:
                - nvidia-cdi-hook
                - chmod
                - --mode
                - "755"
                - --path
                - /dev/nvidia-caps
              env:
                - NVIDIA_CTK_DEBUG=false
containerEdits:
    env:
        - NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu
        - NVIDIA_VISIBLE_DEVICES=void
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
    mounts:
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/libcuda.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/libcuda.so.999.88.77
          options:
            - ro
            - nosuid
            - nodev
            - rbind
            - rprivate
        - hostPath: {{ .driverRoot }}/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
          containerPath: /lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77
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
		// Apply overrides for all test cases:
		tc.options.nvidiaCDIHookPath = "/usr/bin/nvidia-cdi-hook"
		if tc.options.deviceIDs == nil {
			tc.options.deviceIDs = []string{"all"}
			tc.expectedOptions.deviceIDs = []string{"all"}
		}

		t.Run(tc.description, func(t *testing.T) {
			c := command{
				logger: logger,
			}

			err := c.validateFlags(nil, &tc.options)
			require.ErrorIs(t, err, tc.expectedValidateError)
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

			specs, err := c.generateSpecs(&tc.options)
			require.EqualValues(t, err, tc.expectedError)

			var buf bytes.Buffer
			for _, spec := range specs {
				_, err = spec.WriteTo(&buf)
				require.NoError(t, err)
			}

			require.Equal(t, strings.ReplaceAll(tc.expectedSpec, "{{ .driverRoot }}", driverRoot), buf.String())
		})
	}
}

func TestSplitOnAnnotation(t *testing.T) {
	testCases := []struct {
		description            string
		input                  deviceSpecs
		annotation             string
		expectedInputPostSplit deviceSpecs
		expectedOutput         map[string][]specs.Device
	}{
		{
			description: "non-matching annotation",
			input: deviceSpecs{
				specs.Device{
					Name: "foo",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			},
			annotation: "not-key",
			expectedInputPostSplit: deviceSpecs{
				specs.Device{
					Name: "foo",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			},
			expectedOutput: map[string][]specs.Device{},
		},
		{
			description: "single annotation present",
			input: deviceSpecs{
				specs.Device{
					Name: "foo",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			},
			annotation: "key",
			expectedInputPostSplit: deviceSpecs{
				specs.Device{
					Name:        "foo",
					Annotations: map[string]string{},
				},
			},
			expectedOutput: map[string][]specs.Device{
				"key=value": {
					{
						Name:        "foo",
						Annotations: map[string]string{},
					},
				},
			},
		},
		{
			description: "non-matching annotations are not removed",
			input: deviceSpecs{
				specs.Device{
					Name: "foo",
					Annotations: map[string]string{
						"key":     "value",
						"another": "foo",
					},
				},
			},
			annotation: "key",
			expectedInputPostSplit: deviceSpecs{
				specs.Device{
					Name:        "foo",
					Annotations: map[string]string{"another": "foo"},
				},
			},
			expectedOutput: map[string][]specs.Device{
				"key=value": {
					{
						Name:        "foo",
						Annotations: map[string]string{"another": "foo"},
					},
				},
			},
		},
		{
			description: "duplicated annotations different names",
			input: func() deviceSpecs {
				annotations := map[string]string{
					"key": "value",
				}
				return deviceSpecs{
					{Name: "0", Annotations: annotations},
					{Name: "first", Annotations: annotations},
				}
			}(),
			annotation: "key",
			expectedInputPostSplit: deviceSpecs{
				specs.Device{Name: "0", Annotations: map[string]string{}},
				specs.Device{Name: "first", Annotations: map[string]string{}},
			},
			expectedOutput: map[string][]specs.Device{
				"key=value": {
					{Name: "0", Annotations: map[string]string{}},
					{Name: "first", Annotations: map[string]string{}},
				},
			},
		},
		{
			description: "annotation with different values",
			input: deviceSpecs{
				specs.Device{
					Name: "foo",
					Annotations: map[string]string{
						"key": "value",
					},
				},
				specs.Device{
					Name: "bar",
					Annotations: map[string]string{
						"key": "another",
					},
				},
			},
			annotation: "key",
			expectedInputPostSplit: deviceSpecs{
				specs.Device{
					Name:        "foo",
					Annotations: map[string]string{},
				},
				specs.Device{
					Name:        "bar",
					Annotations: map[string]string{},
				},
			},
			expectedOutput: map[string][]specs.Device{
				"key=value": {
					{
						Name:        "foo",
						Annotations: map[string]string{},
					},
				},
				"key=another": {
					{
						Name:        "bar",
						Annotations: map[string]string{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := tc.input.splitOnAnnotation(tc.annotation)

			require.EqualValues(t, tc.expectedOutput, result)
			require.EqualValues(t, tc.expectedInputPostSplit, tc.input)
		})
	}
}
