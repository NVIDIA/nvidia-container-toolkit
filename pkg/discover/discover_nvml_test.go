/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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

package discover

import (
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvml"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/proc"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

const (
	nvidiaGPUDeviceMajorDefault  = 195
	nvidiaCapsDeviceMajorDefault = 235
)

func newTestDiscover() nvmlDiscover {
	logger, _ := testlog.NewNullLogger()
	nvml := nvml.NewMockNVMLServer(nvml.NewMockA100Device(0))

	return nvmlDiscover{
		logger: logger,
		nvml:   nvml,
		nvidiaDevices: proc.NewMockNvidiaDevices(
			proc.Device{Name: nvidiaGPUDeviceName, Major: nvidiaGPUDeviceMajorDefault},
		),
	}
}

func newTestDiscoverWithMIG() nvmlDiscover {
	device := &nvml.MockA100Device{
		Index:              0,
		MigMode:            nvml.DEVICE_MIG_ENABLE,
		GpuInstances:       make(map[*nvml.MockA100GpuInstance]struct{}),
		GpuInstanceCounter: 0,
	}
	// Create a single gi and ci on the device
	gpuInstanceProfileInfo := &nvml.GpuInstanceProfileInfo{
		Id: nvml.GPU_INSTANCE_PROFILE_7_SLICE,
	}
	gi, _ := device.CreateGpuInstance(gpuInstanceProfileInfo)

	computeInstanceProfileInfo := &nvml.ComputeInstanceProfileInfo{
		Id: nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
	}
	_, _ = gi.CreateComputeInstance(computeInstanceProfileInfo)

	logger, _ := testlog.NewNullLogger()
	return nvmlDiscover{
		logger: logger,
		nvml:   nvml.NewMockNVMLServer(device),
		migCaps: map[ProcPath]DeviceNode{
			ProcPath("/proc/driver/nvidia/capabilities/gpu0/mig/gi0/access"): {
				Path:  DevicePath("/dev/nvidia-caps/nvidia-cap3"),
				Minor: 3,
				Major: 235,
			},
			ProcPath("/proc/driver/nvidia/capabilities/gpu0/mig/gi0/ci0/access"): {
				Path:  DevicePath("/dev/nvidia-caps/nvidia-cap4"),
				Minor: 4,
				Major: 235,
			},
			ProcPath("/proc/driver/nvidia/capabilities/mig/config"): {
				Path:  DevicePath("/dev/nvidia-caps/nvidia-cap1"),
				Minor: 1,
				Major: 235,
			},
			ProcPath("/proc/driver/nvidia/capabilities/mig/monitor"): {
				Path:  DevicePath("/dev/nvidia-caps/nvidia-cap2"),
				Minor: 2,
				Major: 235,
			},
		},
		nvidiaDevices: proc.NewMockNvidiaDevices(
			proc.Device{Name: nvidiaGPUDeviceName, Major: nvidiaGPUDeviceMajorDefault},
			proc.Device{Name: nvidiaCapsDeviceName, Major: nvidiaCapsDeviceMajorDefault},
		),
	}

}

func TestDiscoverNvmlDevices(t *testing.T) {
	d := newTestDiscover()
	devices, err := d.Devices()

	require.NoError(t, err)
	require.Len(t, devices, 3)

	device := devices[0]

	require.Equal(t, "0", device.Index)
	require.Equal(t, "GPU-0", device.UUID)
	require.Equal(t, "0000FFFF:FF:FF.F", device.PCIBusID.String())

	expectedDeviceNodes := []DeviceNode{
		{
			Path:  DevicePath("/dev/nvidia0"),
			Minor: 0,
			Major: nvidiaGPUDeviceMajorDefault,
		},
	}
	require.Equal(t, expectedDeviceNodes, device.DeviceNodes)

	expectedProcPaths := []ProcPath{ProcPath("/proc/driver/nvidia/gpus/ffff:ff:ff.f")}
	require.Equal(t, expectedProcPaths, device.ProcPaths)
}

func TestDiscoverNvmlMigDevices(t *testing.T) {
	d := newTestDiscoverWithMIG()

	devices, err := d.Devices()

	require.NoError(t, err)
	require.Len(t, devices, 5)

	mig := devices[0]

	require.Equal(t, "0:0", mig.Index)
	require.Equal(t, "MIG-0", mig.UUID)
	require.Empty(t, mig.PCIBusID)

	expectedDeviceNodes := []DeviceNode{
		{
			Path:  DevicePath("/dev/nvidia0"),
			Minor: 0,
			Major: nvidiaGPUDeviceMajorDefault,
		},
		{
			Path:  DevicePath("/dev/nvidia-caps/nvidia-cap3"),
			Minor: 3,
			Major: nvidiaCapsDeviceMajorDefault,
		},
		{
			Path:  DevicePath("/dev/nvidia-caps/nvidia-cap4"),
			Minor: 4,
			Major: nvidiaCapsDeviceMajorDefault,
		},
	}
	require.Equal(t, expectedDeviceNodes, mig.DeviceNodes)

	expectedProcPaths := []ProcPath{
		ProcPath("/proc/driver/nvidia/gpus/ffff:ff:ff.f"),
		ProcPath("/proc/driver/nvidia/capabilities/gpu0/mig/gi0/access"),
		ProcPath("/proc/driver/nvidia/capabilities/gpu0/mig/gi0/ci0/access"),
	}
	require.Equal(t, expectedProcPaths, mig.ProcPaths)

	var config *Device
	var monitor *Device
	for i, d := range devices {
		if d.UUID == "CONFIG" {
			config = &devices[i]
		}
		if d.UUID == "MONITOR" {
			monitor = &devices[i]
		}
	}

	require.NotNil(t, config)
	require.NotNil(t, monitor)

	require.Equal(t, "CONFIG", config.UUID)

	expectedDeviceNodes = []DeviceNode{
		{
			Path:  DevicePath("/dev/nvidia-caps/nvidia-cap1"),
			Minor: 1,
			Major: nvidiaCapsDeviceMajorDefault,
		},
	}
	require.Equal(t, expectedDeviceNodes, config.DeviceNodes)

	require.Contains(t, config.ProcPaths, ProcPath("/proc/driver/nvidia/capabilities/mig/config"))
	require.Len(t, config.ProcPaths, 2)

	require.Equal(t, "MONITOR", monitor.UUID)

	expectedDeviceNodes = []DeviceNode{
		{
			Path:  DevicePath("/dev/nvidia-caps/nvidia-cap2"),
			Minor: 2,
			Major: nvidiaCapsDeviceMajorDefault,
		},
	}
	require.Equal(t, expectedDeviceNodes, monitor.DeviceNodes)

	require.Contains(t, monitor.ProcPaths, ProcPath("/proc/driver/nvidia/capabilities/mig/monitor"))
	require.Len(t, monitor.ProcPaths, 2)

}

func TestPCIBusID(t *testing.T) {
	testCases := map[string]ProcPath{
		"0000FFFF:FF:FF.F": "/proc/driver/nvidia/gpus/ffff:ff:ff.f",
		"FFFFFFFF:FF:FF.F": "/proc/driver/nvidia/gpus/ffffffff:ff:ff.f",
	}

	for busID, procPath := range testCases {
		p := PCIBusID(busID)
		require.Equal(t, busID, p.String())
		require.Equal(t, procPath, p.GetProcPath())
	}
}
