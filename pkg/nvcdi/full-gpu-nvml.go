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

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/dgpu"
)

// A fullGPUDeviceSpecGenerator generates the CDI device specifications for a
// single full GPU.
type fullGPUDeviceSpecGenerator struct {
	*nvmllib
	id     string
	index  int
	device device.Device
}

var _ DeviceSpecGenerator = (*fullGPUDeviceSpecGenerator)(nil)

func (l *nvmllib) newFullGPUDeviceSpecGeneratorFromNVMLDevice(id string, nvmlDevice nvml.Device) (DeviceSpecGenerator, error) {
	device, err := l.devicelib.NewDevice(nvmlDevice)
	if err != nil {
		return nil, err
	}

	index, ret := nvmlDevice.GetIndex()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device index: %v", ret)
	}

	e := &fullGPUDeviceSpecGenerator{
		nvmllib: l,
		id:      id,
		index:   index,
		device:  device,
	}
	return e, nil
}

func (l *fullGPUDeviceSpecGenerator) GetDeviceSpecs() ([]specs.Device, error) {
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

// GetGPUDeviceEdits returns the CDI edits for the full GPU represented by 'device'.
func (l *fullGPUDeviceSpecGenerator) getDeviceEdits() (*cdi.ContainerEdits, error) {
	device, err := l.newFullGPUDiscoverer(l.device)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	editsForDevice, err := edits.FromDiscoverer(device)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for device: %v", err)
	}

	return editsForDevice, nil
}

func (l *fullGPUDeviceSpecGenerator) getNames() ([]string, error) {
	return l.deviceNamers.GetDeviceNames(l.index, convert{l.device})
}

// newFullGPUDiscoverer creates a discoverer for the full GPU defined by the specified device.
func (l *fullGPUDeviceSpecGenerator) newFullGPUDiscoverer(d device.Device) (discover.Discover, error) {
	deviceNodes, err := dgpu.NewForDevice(d,
		dgpu.WithDevRoot(l.devRoot),
		dgpu.WithLogger(l.logger),
		dgpu.WithHookCreator(l.hookCreator),
		dgpu.WithNvsandboxuitilsLib(l.nvsandboxutilslib),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	deviceFolderPermissionHooks := newDeviceFolderPermissionHookDiscoverer(
		l.logger,
		l.devRoot,
		l.hookCreator,
		deviceNodes,
	)

	dd := discover.Merge(
		deviceNodes,
		deviceFolderPermissionHooks,
	)

	return dd, nil
}
