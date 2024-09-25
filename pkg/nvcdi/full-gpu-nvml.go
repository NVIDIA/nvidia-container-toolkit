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
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/dgpu"
)

// GetGPUDeviceSpecs returns the CDI device specs for the full GPU represented by 'device'.
func (l *nvmllib) GetGPUDeviceSpecs(i int, d device.Device) ([]specs.Device, error) {
	edits, err := l.GetGPUDeviceEdits(d)
	if err != nil {
		return nil, fmt.Errorf("failed to get edits for device: %v", err)
	}

	var deviceSpecs []specs.Device
	names, err := l.deviceNamers.GetDeviceNames(i, convert{d})
	if err != nil {
		return nil, fmt.Errorf("failed to get device name: %v", err)
	}
	for _, name := range names {
		spec := specs.Device{
			Name:           name,
			ContainerEdits: *edits.ContainerEdits,
		}
		deviceSpecs = append(deviceSpecs, spec)
	}

	return deviceSpecs, nil
}

// GetGPUDeviceEdits returns the CDI edits for the full GPU represented by 'device'.
func (l *nvmllib) GetGPUDeviceEdits(d device.Device) (*cdi.ContainerEdits, error) {
	device, err := l.newFullGPUDiscoverer(d)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	editsForDevice, err := edits.FromDiscoverer(device)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for device: %v", err)
	}

	return editsForDevice, nil
}

// newFullGPUDiscoverer creates a discoverer for the full GPU defined by the specified device.
func (l *nvmllib) newFullGPUDiscoverer(d device.Device) (discover.Discover, error) {
	deviceNodes, err := dgpu.NewForDevice(d,
		dgpu.WithDevRoot(l.devRoot),
		dgpu.WithLogger(l.logger),
		dgpu.WithNVIDIACDIHookPath(l.nvidiaCDIHookPath),
		dgpu.WithNvsandboxuitilsLib(l.nvsandboxutilslib),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	deviceFolderPermissionHooks := newDeviceFolderPermissionHookDiscoverer(
		l.logger,
		l.devRoot,
		l.nvidiaCDIHookPath,
		deviceNodes,
	)

	dd := discover.Merge(
		deviceNodes,
		deviceFolderPermissionHooks,
	)

	return dd, nil
}
