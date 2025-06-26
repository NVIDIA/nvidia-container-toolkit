/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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
	"fmt"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/dgpu"
)

type migDeviceSpecGenerator struct {
	*fullGPUDeviceSpecGenerator
	migIndex  int
	migDevice device.MigDevice
}

var _ DeviceSpecGenerator = (*migDeviceSpecGenerator)(nil)

func (l *nvmllib) newMIGDeviceSpecGeneratorFromNVMLDevice(id string, nvmlMIGDevice nvml.Device) (DeviceSpecGenerator, error) {
	nvmlParentDevice, ret := nvmlMIGDevice.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get parent device handle: %v", ret)
	}

	fullGPUGenerator, err := l.newFullGPUDeviceSpecGeneratorFromNVMLDevice(id, nvmlParentDevice)
	if err != nil {
		return nil, err
	}

	nvlibMIGDevice, err := l.devicelib.NewMigDevice(nvmlMIGDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to construct device: %w", err)
	}

	migDeviceIndex, ret := nvmlMIGDevice.GetIndex()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get MIG device index: %w", ret)
	}

	e := &migDeviceSpecGenerator{
		fullGPUDeviceSpecGenerator: fullGPUGenerator.(*fullGPUDeviceSpecGenerator),
		migIndex:                   migDeviceIndex,
		migDevice:                  nvlibMIGDevice,
	}
	return e, nil
}

func (l *migDeviceSpecGenerator) GetDeviceSpecs() ([]specs.Device, error) {
	deviceEdits, err := l.getDeviceEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to get CDI device edits for identifier %q: %w", l.id, err)
	}

	names, err := l.getNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get device names: %w", err)
	}

	var deviceSpecs []specs.Device
	for _, name := range names {
		deviceSpec := specs.Device{
			Name:           name,
			ContainerEdits: *deviceEdits.ContainerEdits,
		}
		deviceSpecs = append(deviceSpecs, deviceSpec)
	}

	return deviceSpecs, nil
}

// GetMIGDeviceEdits returns the CDI edits for the MIG device represented by 'mig' on 'parent'.
func (l *migDeviceSpecGenerator) getDeviceEdits() (*cdi.ContainerEdits, error) {
	deviceNodes, err := dgpu.NewForMigDevice(l.device, l.migDevice,
		dgpu.WithDevRoot(l.devRoot),
		dgpu.WithLogger(l.logger),
		dgpu.WithHookCreator(l.hookCreator),
		dgpu.WithNvsandboxuitilsLib(l.nvsandboxutilslib),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	editsForDevice, err := edits.FromDiscoverer(deviceNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for Compute Instance: %v", err)
	}

	return editsForDevice, nil
}

func (l *migDeviceSpecGenerator) getNames() ([]string, error) {
	return l.deviceNamers.GetMigDeviceNames(l.index, convert{l.device}, l.migIndex, convert{l.migDevice})
}
