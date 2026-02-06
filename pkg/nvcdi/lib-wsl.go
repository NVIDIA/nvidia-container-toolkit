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
)

type wsllib nvcdilib

var _ deviceSpecGeneratorFactory = (*wsllib)(nil)

func (l *wsllib) DeviceSpecGenerators(...string) (DeviceSpecGenerator, error) {
	return l, nil
}

// GetDeviceSpecs returns the CDI device specs for a single all device.
func (l *wsllib) GetDeviceSpecs() ([]specs.Device, error) {
	device := newDXGDeviceDiscoverer(l.logger, l.devRoot)
	deviceEdits, err := l.editsFactory.FromDiscoverer(device)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for DXG device: %v", err)
	}

	deviceSpec := specs.Device{
		Name:           "all",
		ContainerEdits: *deviceEdits.ContainerEdits,
	}

	return []specs.Device{deviceSpec}, nil
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *wsllib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	driver, err := newWSLDriverDiscoverer(l.logger, l.driverRoot, l.hookCreator, l.ldconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for WSL driver: %v", err)
	}

	return l.editsFactory.FromDiscoverer(driver)
}
