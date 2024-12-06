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
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

type nvmllib nvcdilib

var _ Interface = (*nvmllib)(nil)

// GetSpec should not be called for nvmllib
func (l *nvmllib) GetSpec() (spec.Interface, error) {
	return nil, fmt.Errorf("unexpected call to nvmllib.GetSpec()")
}

// GetAllDeviceSpecs returns the device specs for all available devices.
func (l *nvmllib) GetAllDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device

	if r := l.nvmllib.Init(); r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to initialize NVML: %v", r)
	}
	defer func() {
		if r := l.nvmllib.Shutdown(); r != nvml.SUCCESS {
			l.logger.Warningf("failed to shutdown NVML: %v", r)
		}
	}()

	if l.nvsandboxutilslib != nil {
		if r := l.nvsandboxutilslib.Init(l.driverRoot); r != nvsandboxutils.SUCCESS {
			l.logger.Warningf("Failed to init nvsandboxutils: %v; ignoring", r)
			l.nvsandboxutilslib = nil
		}
		defer func() {
			if l.nvsandboxutilslib == nil {
				return
			}
			_ = l.nvsandboxutilslib.Shutdown()
		}()
	}

	gpuDeviceSpecs, err := l.getGPUDeviceSpecs()
	if err != nil {
		return nil, err
	}
	deviceSpecs = append(deviceSpecs, gpuDeviceSpecs...)

	migDeviceSpecs, err := l.getMigDeviceSpecs()
	if err != nil {
		return nil, err
	}
	deviceSpecs = append(deviceSpecs, migDeviceSpecs...)

	return deviceSpecs, nil
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *nvmllib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	common, err := l.newCommonNVMLDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for common entities: %v", err)
	}

	return edits.FromDiscoverer(common)
}

// GetDeviceSpecsByID returns the CDI device specs for the GPU(s) represented by
// the provided identifiers, where an identifier is an index or UUID of a valid
// GPU device.
// Deprecated: Use GetDeviceSpecsBy instead.
func (l *nvmllib) GetDeviceSpecsByID(ids ...string) ([]specs.Device, error) {
	var identifiers []device.Identifier
	for _, id := range ids {
		identifiers = append(identifiers, device.Identifier(id))
	}
	return l.GetDeviceSpecsBy(identifiers...)
}

// GetDeviceSpecsBy returns the device specs for devices with the specified identifiers.
func (l *nvmllib) GetDeviceSpecsBy(identifiers ...device.Identifier) ([]specs.Device, error) {
	for _, id := range identifiers {
		if id == "all" {
			return l.GetAllDeviceSpecs()
		}
	}

	var deviceSpecs []specs.Device

	if r := l.nvmllib.Init(); r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to initialize NVML: %w", r)
	}
	defer func() {
		if r := l.nvmllib.Shutdown(); r != nvml.SUCCESS {
			l.logger.Warningf("failed to shutdown NVML: %v", r)
		}
	}()

	if l.nvsandboxutilslib != nil {
		if r := l.nvsandboxutilslib.Init(l.driverRoot); r != nvsandboxutils.SUCCESS {
			l.logger.Warningf("Failed to init nvsandboxutils: %v; ignoring", r)
			l.nvsandboxutilslib = nil
		}
		defer func() {
			if l.nvsandboxutilslib == nil {
				return
			}
			_ = l.nvsandboxutilslib.Shutdown()
		}()
	}

	nvmlDevices, err := l.getNVMLDevicesByID(identifiers...)
	if err != nil {
		return nil, fmt.Errorf("failed to get NVML device handles: %w", err)
	}

	for i, nvmlDevice := range nvmlDevices {
		deviceEdits, err := l.getEditsForDevice(nvmlDevice)
		if err != nil {
			return nil, fmt.Errorf("failed to get CDI device edits for identifier %q: %w", identifiers[i], err)
		}
		deviceSpec := specs.Device{
			Name:           string(identifiers[i]),
			ContainerEdits: *deviceEdits.ContainerEdits,
		}
		deviceSpecs = append(deviceSpecs, deviceSpec)
	}

	return deviceSpecs, nil
}

// TODO: move this to go-nvlib?
func (l *nvmllib) getNVMLDevicesByID(identifiers ...device.Identifier) ([]nvml.Device, error) {
	var devices []nvml.Device
	for _, id := range identifiers {
		dev, err := l.getNVMLDeviceByID(id)
		if err != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get NVML device handle for identifier %q: %w", id, err)
		}
		devices = append(devices, dev)
	}
	return devices, nil
}

func (l *nvmllib) getNVMLDeviceByID(id device.Identifier) (nvml.Device, error) {
	var err error

	if id.IsUUID() {
		return l.nvmllib.DeviceGetHandleByUUID(string(id))
	}

	if id.IsGpuIndex() {
		if idx, err := strconv.Atoi(string(id)); err == nil {
			return l.nvmllib.DeviceGetHandleByIndex(idx)
		}
		return nil, fmt.Errorf("failed to convert device index to an int: %w", err)
	}

	if id.IsMigIndex() {
		var gpuIdx, migIdx int
		var parent nvml.Device
		split := strings.SplitN(string(id), ":", 2)
		if gpuIdx, err = strconv.Atoi(split[0]); err != nil {
			return nil, fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		if migIdx, err = strconv.Atoi(split[1]); err != nil {
			return nil, fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		if parent, err = l.nvmllib.DeviceGetHandleByIndex(gpuIdx); err != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get parent device handle: %w", err)
		}
		return parent.GetMigDeviceHandleByIndex(migIdx)
	}

	return nil, fmt.Errorf("identifier is not a valid UUID or index: %q", id)
}

func (l *nvmllib) getEditsForDevice(nvmlDevice nvml.Device) (*cdi.ContainerEdits, error) {
	mig, err := nvmlDevice.IsMigDeviceHandle()
	if err != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to determine if device handle is a MIG device: %w", err)
	}
	if mig {
		return l.getEditsForMIGDevice(nvmlDevice)
	}
	return l.getEditsForGPUDevice(nvmlDevice)
}

func (l *nvmllib) getEditsForGPUDevice(nvmlDevice nvml.Device) (*cdi.ContainerEdits, error) {
	nvlibDevice, err := l.devicelib.NewDevice(nvmlDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to construct device: %w", err)
	}
	deviceEdits, err := l.GetGPUDeviceEdits(nvlibDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU device edits: %w", err)
	}

	return deviceEdits, nil
}

func (l *nvmllib) getEditsForMIGDevice(nvmlDevice nvml.Device) (*cdi.ContainerEdits, error) {
	nvmlParentDevice, ret := nvmlDevice.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get parent device handle: %w", ret)
	}
	nvlibMigDevice, err := l.devicelib.NewMigDevice(nvmlDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to construct device: %w", err)
	}
	nvlibParentDevice, err := l.devicelib.NewDevice(nvmlParentDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to construct parent device: %w", err)
	}
	return l.GetMIGDeviceEdits(nvlibParentDevice, nvlibMigDevice)
}

func (l *nvmllib) getGPUDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device
	err := l.devicelib.VisitDevices(func(i int, d device.Device) error {
		specsForDevice, err := l.GetGPUDeviceSpecs(i, d)
		if err != nil {
			return err
		}
		deviceSpecs = append(deviceSpecs, specsForDevice...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI edits for GPU devices: %v", err)
	}
	return deviceSpecs, err
}

func (l *nvmllib) getMigDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device
	err := l.devicelib.VisitMigDevices(func(i int, d device.Device, j int, mig device.MigDevice) error {
		specsForDevice, err := l.GetMIGDeviceSpecs(i, d, j, mig)
		if err != nil {
			return err
		}
		deviceSpecs = append(deviceSpecs, specsForDevice...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI edits for GPU devices: %v", err)
	}
	return deviceSpecs, err
}
