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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra"
)

type csvlib nvcdilib

var _ deviceSpecGeneratorFactory = (*csvlib)(nil)

func (l *csvlib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	for _, id := range ids {
		switch id {
		case "all":
		case "0":
		default:
			return nil, fmt.Errorf("unsupported device id: %v", id)
		}
	}

	return l, nil
}

// GetDeviceSpecs returns the CDI device specs for a single device.
func (l *csvlib) GetDeviceSpecs() ([]specs.Device, error) {
	d, err := tegra.New(
		tegra.WithLogger(l.logger),
		tegra.WithDriverRoot(l.driverRoot),
		tegra.WithDevRoot(l.devRoot),
		tegra.WithHookCreator(l.hookCreator),
		tegra.WithLdconfigPath(l.ldconfigPath),
		tegra.WithCSVFiles(l.csvFiles),
		tegra.WithLibrarySearchPaths(l.librarySearchPaths...),
		tegra.WithIngorePatterns(l.csvIgnorePatterns...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for CSV files: %v", err)
	}
	e, err := edits.FromDiscoverer(d)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for CSV files: %v", err)
	}

	names, err := l.deviceNamers.GetDeviceNames(0, uuidIgnored{})
	if err != nil {
		return nil, fmt.Errorf("failed to get device name: %v", err)
	}
	var deviceSpecs []specs.Device
	for _, name := range names {
		deviceSpec := specs.Device{
			Name:           name,
			ContainerEdits: *e.ContainerEdits,
		}
		deviceSpecs = append(deviceSpecs, deviceSpec)
	}

	return deviceSpecs, nil
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *csvlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return edits.FromDiscoverer(discover.None{})
}
