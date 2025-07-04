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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
)

type mofedlib nvcdilib

var _ deviceSpecGeneratorFactory = (*mofedlib)(nil)

func (l *mofedlib) DeviceSpecGenerators(...string) (DeviceSpecGenerator, error) {
	return l, nil
}

// GetDeviceSpecs returns the CDI device specs for a single all device.
func (l *mofedlib) GetDeviceSpecs() ([]specs.Device, error) {
	discoverer, err := discover.NewMOFEDDiscoverer(l.logger, l.driverRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create MOFED discoverer: %v", err)
	}
	edits, err := edits.FromDiscoverer(discoverer)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for MOFED devices: %v", err)
	}

	deviceSpec := specs.Device{
		Name:           "all",
		ContainerEdits: *edits.ContainerEdits,
	}

	return []specs.Device{deviceSpec}, nil
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *mofedlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return edits.FromDiscoverer(discover.None{})
}
