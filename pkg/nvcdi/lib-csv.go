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
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

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
	d, err := l.driverDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create driver discoverer from CSV files: %w", err)
	}
	e, err := edits.FromDiscoverer(d)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for CSV files: %w", err)
	}

	names, err := l.deviceNamers.GetDeviceNames(0, uuidIgnored{})
	if err != nil {
		return nil, fmt.Errorf("failed to get device name: %w", err)
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

func (l *csvlib) driverDiscoverer() (discover.Discover, error) {
	driverDiscoverer, err := tegra.New(
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
		return nil, fmt.Errorf("failed to create discoverer for CSV files: %w", err)
	}

	cudaCompatDiscoverer := l.cudaCompatDiscoverer()

	ldcacheUpdateHook, err := discover.NewLDCacheUpdateHook(l.logger, driverDiscoverer, l.hookCreator, l.ldconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ldcache update hook discoverer: %w", err)
	}

	d := discover.Merge(
		driverDiscoverer,
		cudaCompatDiscoverer,
		// The ldcacheUpdateHook is added last to ensure that the created symlinks are included
		ldcacheUpdateHook,
	)
	return d, nil
}

func (l *csvlib) cudaCompatDiscoverer() discover.Discover {
	hasNvml, _ := l.infolib.HasNvml()
	if !hasNvml {
		return nil
	}

	ret := l.nvmllib.Init()
	if ret != nvml.SUCCESS {
		l.logger.Warningf("Failed to initialize NVML: %v", ret)
		return nil
	}
	defer func() {
		_ = l.nvmllib.Shutdown()
	}()

	version, ret := l.nvmllib.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		l.logger.Warningf("Failed to get driver version: %v", ret)
		return nil
	}

	var names []string
	err := l.devicelib.VisitDevices(func(i int, d device.Device) error {
		name, ret := d.GetName()
		if ret != nvml.SUCCESS {
			return fmt.Errorf("device %v: %v", i, ret)
		}
		names = append(names, name)
		return nil
	})
	if err != nil {
		l.logger.Warningf("Failed to get device names: %v", err)
	}

	var cudaCompatContainerRoot string
	for _, name := range names {
		if strings.Contains(name, "Orin (nvgpu)") {
			// TODO: This should probably be a constant.
			cudaCompatContainerRoot = "/usr/local/cuda/compat-orin"
			break
		}
	}

	return discover.NewCUDACompatHookDiscoverer(l.logger, l.hookCreator, version, cudaCompatContainerRoot)
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *csvlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return edits.FromDiscoverer(discover.None{})
}
