/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package nvcdi

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestDeviceSpecGenerators(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	logger, _ := testlog.NewNullLogger()

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testCases := []struct {
		description string

		rootfsFolder string

		lib                 *csvlib
		expectedError       error
		expectedSpecError   error
		expectedDeviceSpecs []specs.Device
		expectedCommonEdits *cdi.ContainerEdits
	}{
		{
			description:  "single orin CSV device",
			rootfsFolder: "rootfs-orin",
			lib: &csvlib{
				// test-case specific
				infolib: &infoInterfaceMock{
					HasNvmlFunc: func() (bool, string) { return true, "forced" },
				},
				// TODO: Replace this with a system-specific implementation once available.
				nvmllib: &mock.Interface{
					InitFunc: func() nvml.Return {
						return nvml.SUCCESS
					},
					ShutdownFunc: func() nvml.Return {
						return nvml.SUCCESS
					},
					SystemGetDriverVersionFunc: func() (string, nvml.Return) {
						return "540.3.0", nvml.SUCCESS
					},
					DeviceGetCountFunc: func() (int, nvml.Return) {
						return 1, nvml.SUCCESS
					},
					DeviceGetHandleByIndexFunc: func(n int) (nvml.Device, nvml.Return) {
						if n != 0 {
							return nil, nvml.ERROR_INVALID_ARGUMENT
						}
						device := &mock.Device{
							GetUUIDFunc: func() (string, nvml.Return) {
								return "GPU-orin", nvml.SUCCESS
							},
							GetNameFunc: func() (string, nvml.Return) {
								return "Orin (nvgpu)", nvml.SUCCESS
							},
							GetPciInfoFunc: func() (nvml.PciInfo, nvml.Return) {
								return nvml.PciInfo{}, nvml.ERROR_NOT_SUPPORTED
							},
						}
						return device, nvml.SUCCESS
					},
				},
			},
			expectedDeviceSpecs: []specs.Device{
				{
					Name: "0",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
						},
					},
				},
			},
			expectedCommonEdits: &cdi.ContainerEdits{
				ContainerEdits: &specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "enable-cuda-compat", "--host-driver-version=540.3.0", "--cuda-compat-container-root=/usr/local/cuda/compat-orin"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "update-ldcache"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
					},
				},
			},
		},
		{
			description:  "thor device with dGPU",
			rootfsFolder: "rootfs-thor-dgpu",
			lib: &csvlib{
				// test-case specific
				infolib: &infoInterfaceMock{
					HasNvmlFunc: func() (bool, string) { return true, "forced" },
				},
				nvmllib: mockIGXServer(),
			},
			expectedDeviceSpecs: []specs.Device{
				{
					Name: "0",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
							{Path: "/dev/nvidiactl", HostPath: "/dev/nvidiactl"},
							{Path: "/dev/nvidia2", HostPath: "/dev/nvidia2"},
						},
					},
				},
				{
					Name: "1",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{Path: "/dev/nvidia1", HostPath: "/dev/nvidia1"},
							{Path: "/dev/nvidiactl", HostPath: "/dev/nvidiactl"},
						},
					},
				},
			},
			expectedCommonEdits: &cdi.ContainerEdits{
				ContainerEdits: &specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "enable-cuda-compat", "--host-driver-version=540.3.0"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "update-ldcache"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		driverRoot := filepath.Join(lookupRoot, tc.rootfsFolder)

		tc.lib.logger = logger
		tc.lib.editsFactory = edits.NewFactory(edits.WithLogger(logger))
		tc.lib.deviceNamers = []DeviceNamer{deviceNameIndex{}}
		tc.lib.hookCreator = discover.NewHookCreator()

		tc.lib.devicelib = device.New(tc.lib.nvmllib)

		tc.lib.driverRoot = driverRoot
		tc.lib.devRoot = driverRoot
		tc.lib.csvFiles = []string{
			filepath.Join(driverRoot, "/etc/nvidia-container-runtime/host-files-for-container.d/devices.csv"),
			filepath.Join(driverRoot, "/etc/nvidia-container-runtime/host-files-for-container.d/drivers.csv"),
		}

		t.Run(tc.description, func(t *testing.T) {
			generator, err := tc.lib.DeviceSpecGenerators("all")

			require.EqualValues(t, tc.expectedError, err)

			if tc.expectedError != nil {
				return
			}

			deviceSpecs, err := generator.GetDeviceSpecs()
			require.EqualValues(t, tc.expectedSpecError, err)
			require.EqualValues(t, tc.expectedDeviceSpecs, stripRoot(driverRoot, deviceSpecs))

			commonEdits, err := tc.lib.GetCommonEdits()
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedCommonEdits, commonEdits)
		})
	}

}

func stripRoot[T any](root string, v T) T {
	stringRep, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	stringRep = bytes.ReplaceAll(stringRep, []byte(root), []byte(""))

	var modified T
	err = json.Unmarshal(stringRep, &modified)
	if err != nil {
		panic(err)
	}
	return modified
}

// TODO: We should move this mock to go-nvml/mock
func mockIGXServer() nvml.Interface {
	thor := &mock.Device{
		GetNameFunc: func() (string, nvml.Return) {
			return "NVIDIA Thor", nvml.SUCCESS
		},
		GetUUIDFunc: func() (string, nvml.Return) {
			return "GPU-0", nvml.SUCCESS
		},
		GetPciInfoFunc: func() (nvml.PciInfo, nvml.Return) {
			return nvml.PciInfo{
				Bus: 1,
			}, nvml.SUCCESS
		},
	}
	rtx := &mock.Device{
		GetNameFunc: func() (string, nvml.Return) {
			return "RTX Pro 6000", nvml.SUCCESS
		},
		GetUUIDFunc: func() (string, nvml.Return) {
			return "GPU-1", nvml.SUCCESS
		},
		GetPciInfoFunc: func() (nvml.PciInfo, nvml.Return) {
			return nvml.PciInfo{
				Bus: 3,
			}, nvml.SUCCESS
		},
		GetMinorNumberFunc: func() (int, nvml.Return) {
			return 1, nvml.SUCCESS
		},
	}

	return &mock.Interface{
		InitFunc: func() nvml.Return {
			return nvml.SUCCESS
		},
		ShutdownFunc: func() nvml.Return {
			return nvml.SUCCESS
		},
		SystemGetDriverVersionFunc: func() (string, nvml.Return) {
			return "540.3.0", nvml.SUCCESS
		},
		DeviceGetCountFunc: func() (int, nvml.Return) {
			return 2, nvml.SUCCESS
		},
		DeviceGetHandleByIndexFunc: func(n int) (nvml.Device, nvml.Return) {
			switch n {
			case 0:
				return thor, nvml.SUCCESS
			case 1:
				return rtx, nvml.SUCCESS
			}
			return nil, nvml.ERROR_INVALID_ARGUMENT
		},
		DeviceGetHandleByUUIDFunc: func(s string) (nvml.Device, nvml.Return) {
			switch s {
			case "GPU-0":
				return thor, nvml.SUCCESS
			case "GPU-1":
				return rtx, nvml.SUCCESS
			}
			return nil, nvml.ERROR_INVALID_ARGUMENT
		},
	}
}
