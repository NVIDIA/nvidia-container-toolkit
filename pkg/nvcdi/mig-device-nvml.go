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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/dgpu"
)

// GetMIGDeviceSpecs returns the CDI device specs for the full GPU represented by 'device'.
func (l *nvmllib) GetMIGDeviceSpecs(i int, d device.Device, j int, mig device.MigDevice) ([]specs.Device, error) {
	edits, err := l.GetMIGDeviceEdits(d, mig)
	if err != nil {
		return nil, fmt.Errorf("failed to get edits for device: %v", err)
	}

	names, err := l.deviceNamers.GetMigDeviceNames(i, convert{d}, j, convert{mig})
	if err != nil {
		return nil, fmt.Errorf("failed to get device name: %v", err)
	}
	var deviceSpecs []specs.Device
	for _, name := range names {
		spec := specs.Device{
			Name:           name,
			ContainerEdits: *edits.ContainerEdits,
		}
		deviceSpecs = append(deviceSpecs, spec)
	}
	return deviceSpecs, nil
}

// GetMIGDeviceEdits returns the CDI edits for the MIG device represented by 'mig' on 'parent'.
func (l *nvmllib) GetMIGDeviceEdits(parent device.Device, mig device.MigDevice) (*cdi.ContainerEdits, error) {
	deviceNodes, err := dgpu.NewForMigDevice(parent, mig,
		dgpu.WithDevRoot(l.devRoot),
		dgpu.WithLogger(l.logger),
		dgpu.WithNVIDIACDIHookPath(l.nvidiaCDIHookPath),
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
