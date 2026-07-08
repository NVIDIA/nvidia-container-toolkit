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
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
)

// TODO: In order to properly test this, we need a mechanism to inject /
// override the char device discoverer.
func TestNewNvmlDGPUDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	nvmllib := &mock.Interface{}
	devicelib := device.New(
		nvmllib,
	)

	testCases := []struct {
		description     string
		device          nvml.Device
		expectedError   error
		expectedDevices []discover.Device
		expectedHooks   []discover.Hook
		expectedMounts  []discover.Mount
	}{
		{
			description: "",
			device: &mock.Device{
				GetMinorNumberFunc: func() (int, nvml.Return) {
					return 3, nvml.SUCCESS
				},
				GetPciInfoFunc: func() (nvml.PciInfo, nvml.Return) {
					var busID [32]int8
					for i, b := range []byte("00000000:45:00:00") {
						busID[i] = int8(b)
					}
					info := nvml.PciInfo{
						BusId: busID,
					}
					return info, nvml.SUCCESS
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			o, err := new(WithLogger(logger), WithDriver(root.New()))
			require.NoError(t, err)

			device, err := devicelib.NewDevice(tc.device)
			require.NoError(t, err)

			d, err := o.newNvmlDGPUDiscoverer(&toRequiredInfo{device})
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

func TestNewNvmlMIGDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	nvmllib := &mock.Interface{}
	devicelib := device.New(
		nvmllib,
	)

	parentMock, migMock := newNvmlMigDiscovererTestMocks()

	testCases := []struct {
		description     string
		mig             *mock.Device
		parent          nvml.Device
		migCaps         nvcaps.MigCaps
		expectedError   error
		expectedDevices []discover.Device
		expectedHooks   []discover.Hook
		expectedMounts  []discover.Mount
	}{
		{
			description: "",
			mig:         migMock,
			parent:      parentMock,
			migCaps: nvcaps.MigCaps{
				"gpu3/gi1/access":     31,
				"gpu3/gi1/ci2/access": 312,
			},
			expectedDevices: nil,
			expectedMounts:  nil,
			expectedHooks:   nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			tc.mig.GetDeviceHandleFromMigDeviceHandleFunc = func() (nvml.Device, nvml.Return) {
				return tc.parent, nvml.SUCCESS
			}
			parent, err := devicelib.NewDevice(tc.parent)
			require.NoError(t, err)

			mig, err := devicelib.NewMigDevice(tc.mig)
			require.NoError(t, err)

			d, err := NewForMigDevice(parent, mig,
				WithLogger(logger),
				WithDriver(root.New()),
				WithMIGCaps(tc.migCaps),
			)
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

// newNvmlMigDiscovererTestMocks returns parent and MIG NVML device mocks wired so
// go-nvlib device.MigDevice.GetProfile succeeds (GPU instance id 1, compute instance id 2).
func newNvmlMigDiscovererTestMocks() (parent *mock.Device, mig *mock.Device) {
	const (
		gpuInstanceID     = 1
		computeInstanceID = 2
	)

	mockCI := &mock.ComputeInstance{
		GetInfoFunc: func() (nvml.ComputeInstanceInfo, nvml.Return) {
			return nvml.ComputeInstanceInfo{
				ProfileId: nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			}, nvml.SUCCESS
		},
	}

	mockGI := &mock.GpuInstance{
		GetInfoFunc: func() (nvml.GpuInstanceInfo, nvml.Return) {
			return nvml.GpuInstanceInfo{
				ProfileId: nvml.GPU_INSTANCE_PROFILE_1_SLICE,
			}, nvml.SUCCESS
		},
		GetComputeInstanceByIdFunc: func(n int) (nvml.ComputeInstance, nvml.Return) {
			if n == computeInstanceID {
				return mockCI, nvml.SUCCESS
			}
			return nil, nvml.ERROR_INVALID_ARGUMENT
		},
		GetComputeInstanceProfileInfoFunc: func(n1, n2 int) (nvml.ComputeInstanceProfileInfo, nvml.Return) {
			if n1 == 0 && n2 == 0 {
				return nvml.ComputeInstanceProfileInfo{
					Id: nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				}, nvml.SUCCESS
			}
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		},
	}

	parent = &mock.Device{
		GetMinorNumberFunc: func() (int, nvml.Return) {
			return 3, nvml.SUCCESS
		},
		GetPciInfoFunc: func() (nvml.PciInfo, nvml.Return) {
			var busID [32]int8
			for i, b := range []byte("00000000:45:00:00") {
				busID[i] = int8(b)
			}
			return nvml.PciInfo{BusId: busID}, nvml.SUCCESS
		},
		GetMemoryInfoFunc: func() (nvml.Memory, nvml.Return) {
			// Non-zero total memory is required for MIG profile memory math in go-nvlib.
			return nvml.Memory{Total: 40 * 1024 * 1024 * 1024}, nvml.SUCCESS
		},
		GetGpuInstanceByIdFunc: func(n int) (nvml.GpuInstance, nvml.Return) {
			if n == gpuInstanceID {
				return mockGI, nvml.SUCCESS
			}
			return nil, nvml.ERROR_INVALID_ARGUMENT
		},
		GetGpuInstanceProfileInfoFunc: func(n int) (nvml.GpuInstanceProfileInfo, nvml.Return) {
			if n == 0 {
				return nvml.GpuInstanceProfileInfo{
					Id:           nvml.GPU_INSTANCE_PROFILE_1_SLICE,
					MemorySizeMB: 5120,
				}, nvml.SUCCESS
			}
			return nvml.GpuInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		},
	}

	mig = &mock.Device{
		IsMigDeviceHandleFunc: func() (bool, nvml.Return) {
			return true, nvml.SUCCESS
		},
		GetGpuInstanceIdFunc: func() (int, nvml.Return) {
			return gpuInstanceID, nvml.SUCCESS
		},
		GetComputeInstanceIdFunc: func() (int, nvml.Return) {
			return computeInstanceID, nvml.SUCCESS
		},
		GetAttributesFunc: func() (nvml.DeviceAttributes, nvml.Return) {
			return nvml.DeviceAttributes{MemorySizeMB: 5120}, nvml.SUCCESS
		},
	}

	return parent, mig
}
