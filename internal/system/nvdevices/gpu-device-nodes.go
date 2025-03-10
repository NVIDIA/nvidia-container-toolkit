/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package nvdevices

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/go-nvlib/pkg/nvpci"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
)

func (m *Interface) createGPUDeviceNode(gpuIndex int) error {
	major, exists := m.Get(devices.NVIDIAGPU)
	if !exists {
		return fmt.Errorf("failed to determine device major; nvidia kernel module may not be loaded")
	}

	deviceNodePath := fmt.Sprintf("/dev/nvidia%d", gpuIndex)
	if err := m.createDeviceNode(deviceNodePath, int(major), gpuIndex); err != nil {
		return fmt.Errorf("failed to create device node %v: %w", deviceNodePath, err)
	}
	return nil
}

func (m *Interface) createMigDeviceNodes(gpuIndex int) error {
	capsMajor, exists := m.Get("nvidia-caps")
	if !exists {
		return nil
	}
	var errs error
	for _, capsDeviceMinor := range m.migCaps.FilterForGPU(gpuIndex) {
		capDevicePath := capsDeviceMinor.DevicePath()
		err := m.createDeviceNode(capDevicePath, int(capsMajor), int(capsDeviceMinor))
		errs = errors.Join(errs, fmt.Errorf("failed to create %v: %w", capDevicePath, err))
	}
	return errs
}

func (m *Interface) createAllGPUDeviceNodes() error {
	gpus, err := nvpci.New(
		nvpci.WithPCIDevicesRoot(filepath.Join(m.devRoot, nvpci.PCIDevicesRoot)),
		nvpci.WithLogger(m.logger),
	).GetGPUs()
	if err != nil {
		return fmt.Errorf("failed to get GPU information from PCI: %w", err)
	}

	count := len(gpus)
	if count == 0 {
		return nil
	}

	var errs error
	for gpuIndex := 0; gpuIndex < count; gpuIndex++ {
		errs = errors.Join(errs, m.createGPUDeviceNode(gpuIndex))
		errs = errors.Join(errs, m.createMigDeviceNodes(gpuIndex))
	}
	return errs
}
