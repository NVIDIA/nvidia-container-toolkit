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
)

type gatedlib nvcdilib

var _ deviceSpecGeneratorFactory = (*gatedlib)(nil)

func (l *gatedlib) DeviceSpecGenerators(...string) (DeviceSpecGenerator, error) {
	return l, nil
}

// GetDeviceSpecs returns the CDI device specs for a single all device.
func (l *gatedlib) GetDeviceSpecs() ([]specs.Device, error) {
	discoverer, err := l.getModeDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for mode %q: %w", l.mode, err)
	}
	edits, err := l.editsFactory.FromDiscoverer(discoverer)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits: %w", err)
	}

	deviceSpec := specs.Device{
		Name:           "all",
		ContainerEdits: *edits.ContainerEdits,
	}

	return []specs.Device{deviceSpec}, nil
}

func (l *gatedlib) getModeDiscoverer() (discover.Discover, error) {
	switch l.mode {
	case ModeGdrcopy:
		return discover.NewGDRCopyDiscoverer(l.logger, l.devRoot)
	case ModeGds:
		return discover.NewGDSDiscoverer(l.logger, l.driverRoot, l.devRoot)
	case ModeMofed:
		return discover.NewMOFEDDiscoverer(l.logger, l.driverRoot)
	case ModeNvswitch:
		return discover.NewNvSwitchDiscoverer(l.logger, l.devRoot)
	default:
		return nil, fmt.Errorf("unrecognized mode")
	}
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *gatedlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return l.editsFactory.FromDiscoverer(discover.None{})
}
