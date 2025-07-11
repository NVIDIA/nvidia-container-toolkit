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

package dgpu

import (
	"testing"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	mocknvml "github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
	mocknvsandboxutils "github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils/mock"
)

func TestNewNvsandboxutilsDGPUDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	nvmllib := &mocknvml.Interface{}
	devicelib := device.New(
		nvmllib,
	)

	testCases := []struct {
		description     string
		devRoot         string
		device          nvml.Device
		nvsandboxutils  nvsandboxutils.Interface
		expectedError   error
		expectedDevices []discover.Device
		expectedHooks   []discover.Hook
		expectedMounts  []discover.Mount
	}{
		{
			description: "detects host devices",
			device: &mocknvml.Device{
				GetUUIDFunc: func() (string, nvml.Return) {
					return "GPU-1234", nvml.SUCCESS
				},
			},
			nvsandboxutils: &mocknvsandboxutils.Interface{
				GetGpuResourceFunc: func(s string) ([]nvsandboxutils.GpuFileInfo, nvsandboxutils.Ret) {
					infos := []nvsandboxutils.GpuFileInfo{
						{
							Path: "/dev/nvidia0",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/dev/nvidiactl",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/dev/nvidia-uvm",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/dev/nvidia-uvm-tools",
							Type: nvsandboxutils.NV_DEV,
						},
					}
					return infos, nvsandboxutils.SUCCESS
				},
			},
			expectedDevices: []discover.Device{
				{
					Path:     "/dev/nvidia0",
					HostPath: "/dev/nvidia0",
				},
				{
					Path:     "/dev/nvidiactl",
					HostPath: "/dev/nvidiactl",
				},
				{
					Path:     "/dev/nvidia-uvm",
					HostPath: "/dev/nvidia-uvm",
				},
				{
					Path:     "/dev/nvidia-uvm-tools",
					HostPath: "/dev/nvidia-uvm-tools",
				},
			},
		},
		{
			description: "detects container devices",
			devRoot:     "/some/root",
			device: &mocknvml.Device{
				GetUUIDFunc: func() (string, nvml.Return) {
					return "GPU-1234", nvml.SUCCESS
				},
			},
			nvsandboxutils: &mocknvsandboxutils.Interface{
				GetGpuResourceFunc: func(s string) ([]nvsandboxutils.GpuFileInfo, nvsandboxutils.Ret) {
					infos := []nvsandboxutils.GpuFileInfo{
						{
							Path: "/some/root/dev/nvidia0",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/some/root/dev/nvidiactl",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/some/root/dev/nvidia-uvm",
							Type: nvsandboxutils.NV_DEV,
						},
						{
							Path: "/some/root/dev/nvidia-uvm-tools",
							Type: nvsandboxutils.NV_DEV,
						},
					}
					return infos, nvsandboxutils.SUCCESS
				},
			},
			expectedDevices: []discover.Device{
				{
					Path:     "/dev/nvidia0",
					HostPath: "/some/root/dev/nvidia0",
				},
				{
					Path:     "/dev/nvidiactl",
					HostPath: "/some/root/dev/nvidiactl",
				},
				{
					Path:     "/dev/nvidia-uvm",
					HostPath: "/some/root/dev/nvidia-uvm",
				},
				{
					Path:     "/dev/nvidia-uvm-tools",
					HostPath: "/some/root/dev/nvidia-uvm-tools",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			o := &options{
				logger:            logger,
				devRoot:           tc.devRoot,
				nvsandboxutilslib: tc.nvsandboxutils,
			}

			device, err := devicelib.NewDevice(tc.device)
			require.NoError(t, err)

			d, err := o.newNvsandboxutilsDGPUDiscoverer(device)
			require.ErrorIs(t, err, tc.expectedError)

			devices, _ := d.Devices()
			require.EqualValues(t, tc.expectedDevices, devices)
			hooks, _ := d.Hooks()
			require.EqualValues(t, tc.expectedHooks, hooks)
			mounts, _ := d.Mounts()
			require.EqualValues(t, tc.expectedMounts, mounts)
		})
	}
}
