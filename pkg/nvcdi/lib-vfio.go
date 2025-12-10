/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	"fmt"
	"path/filepath"
	"strconv"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"
)

type vfiolib nvcdilib

type vfioDevice struct {
	index   int
	group   int
	devRoot string
}

var _ deviceSpecGeneratorFactory = (*vfiolib)(nil)

func (l *vfiolib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	vfioDevices, err := l.getVfioDevices(ids...)
	if err != nil {
		return nil, err
	}
	var deviceSpecGenerators DeviceSpecGenerators
	for _, vfioDevice := range vfioDevices {
		deviceSpecGenerators = append(deviceSpecGenerators, vfioDevice)
	}

	return deviceSpecGenerators, nil
}

// GetDeviceSpecs returns the CDI device specs for a vfio device.
func (l *vfioDevice) GetDeviceSpecs() ([]specs.Device, error) {
	path := fmt.Sprintf("/dev/vfio/%d", l.group)
	deviceSpec := specs.Device{
		Name: fmt.Sprintf("%d", l.index),
		ContainerEdits: specs.ContainerEdits{
			DeviceNodes: []*specs.DeviceNode{
				{
					Path:     path,
					HostPath: filepath.Join(l.devRoot, path),
				},
			},
		},
	}
	return []specs.Device{deviceSpec}, nil
}

// GetCommonEdits returns common edits for ALL devices.
// Note, currently there are no common edits.
func (l *vfiolib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	e := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{
			DeviceNodes: []*specs.DeviceNode{
				{
					Path:     "/dev/vfio/vfio",
					HostPath: filepath.Join(l.devRoot, "/dev/vfio/vfio"),
				},
			},
		},
	}
	return &e, nil
}

func (l *vfiolib) getVfioDevices(ids ...string) ([]*vfioDevice, error) {
	var vfioDevices []*vfioDevice
	for _, id := range ids {
		if id == "all" {
			return l.getAllVfioDevices()
		}
		index, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid channel ID %v: %w", id, err)
		}
		i := int(index)
		dev, err := l.nvpcilib.GetGPUByIndex(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get device: %w", err)
		}
		vfioDevices = append(vfioDevices, &vfioDevice{index: i, group: dev.IommuGroup, devRoot: l.devRoot})
	}

	return vfioDevices, nil
}

func (l *vfiolib) getAllVfioDevices() ([]*vfioDevice, error) {
	devices, err := l.nvpcilib.GetGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed getting NVIDIA GPUs: %v", err)
	}

	var vfioDevices []*vfioDevice
	for i, dev := range devices {
		if dev.Driver != "vfio-pci" {
			continue
		}
		l.logger.Debugf("Found NVIDIA device: address=%s, driver=%s, iommu_group=%d, deviceId=%x",
			dev.Address, dev.Driver, dev.IommuGroup, dev.Device)
		vfioDevices = append(vfioDevices, &vfioDevice{index: i, group: dev.IommuGroup, devRoot: l.devRoot})
	}
	return vfioDevices, nil
}
