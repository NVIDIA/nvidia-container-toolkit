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
)

type nvmllib nvcdilib

var _ deviceSpecGeneratorFactory = (*nvmllib)(nil)

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *nvmllib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	common, err := l.newCommonNVMLDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for common entities: %v", err)
	}

	return edits.FromDiscoverer(common)
}

// DeviceSpecGenerators returns the CDI device spec generators for NVML devices
// with the specified IDs.
// Supported IDs are:
// * an index of a GPU or MIG device
// * a UUID of a GPU or MIG device
// * the special ID 'all'
func (l *nvmllib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	if err := l.init(); err != nil {
		return nil, err
	}
	defer l.tryShutdown()

	dsgs, err := l.getDeviceSpecGeneratorsForIDs(ids...)
	if err != nil {
		return nil, err
	}
	return l.withInit(dsgs), nil
}

func (l *nvmllib) getDeviceSpecGeneratorsForIDs(ids ...string) (DeviceSpecGenerator, error) {
	var identifiers []device.Identifier
	for _, id := range ids {
		if id == "none" {
			return DeviceSpecGenerators{}, nil
		}
		if id == "all" {
			return l.getDeviceSpecGeneratorsForAllDevices()
		}
		identifiers = append(identifiers, device.Identifier(id))
	}

	uuids, err := l.normalizeDeviceIDs(identifiers...)
	if err != nil {
		return nil, err
	}

	var DeviceSpecGenerators DeviceSpecGenerators
	for _, uuid := range uuids {
		device, ret := l.nvmllib.DeviceGetHandleByUUID(string(uuid))
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device handle from UUID: %v", ret)
		}
		generator, err := l.newDeviceSpecGeneratorFromNVMLDevice(string(uuid), device)
		if err != nil {
			return nil, err
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, generator)
	}

	return DeviceSpecGenerators, nil
}

func (l *nvmllib) newDeviceSpecGeneratorFromNVMLDevice(id string, nvmlDevice nvml.Device) (DeviceSpecGenerator, error) {
	isMig, ret := nvmlDevice.IsMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("%v", ret)
	}
	if isMig {
		return l.newMIGDeviceSpecGeneratorFromNVMLDevice(id, nvmlDevice)
	}

	return l.newFullGPUDeviceSpecGeneratorFromNVMLDevice(id, nvmlDevice, l.featureFlags)
}

// getDeviceSpecGeneratorsForAllDevices returns the CDI device spec generators
// for all NVML devices detected on the system.
// This includes full GPUs as well as MIG devices.
func (l *nvmllib) getDeviceSpecGeneratorsForAllDevices() (DeviceSpecGenerator, error) {
	var DeviceSpecGenerators DeviceSpecGenerators
	err := l.devicelib.VisitDevices(func(i int, d device.Device) error {
		isMigEnabled, err := d.IsMigEnabled()
		if err != nil {
			return err
		}
		if isMigEnabled {
			return nil
		}
		fullGPU, err := l.newFullGPUDeviceSpecGeneratorFromDevice(i, d, l.featureFlags)
		if err != nil {
			return err
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, fullGPU)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get full GPU device editors: %w", err)
	}

	err = l.devicelib.VisitMigDevices(func(i int, d device.Device, j int, mig device.MigDevice) error {
		migDevice, err := l.newMIGDeviceSpecGeneratorFromDevice(i, d, j, mig)
		if err != nil {
			return err
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, migDevice)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get MIG device editors: %w", err)
	}

	return DeviceSpecGenerators, nil
}

// TODO: move this to go-nvlib?
// normalizeDeviceID returns the UUIDs of the devices specified by the identifier.
func (l *nvmllib) normalizeDeviceIDs(identifiers ...device.Identifier) ([]device.Identifier, error) {
	var uuids []device.Identifier
	for _, id := range identifiers {
		uuid, err := l.normalizeDeviceID(id)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

func (l *nvmllib) normalizeDeviceID(id device.Identifier) (device.Identifier, error) {
	var err error

	if id.IsUUID() {
		return id, nil
	}

	if id.IsGpuIndex() {
		idx, err := strconv.Atoi(string(id))
		if err != nil {
			return "", fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		dev, ret := l.nvmllib.DeviceGetHandleByIndex(idx)
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("failed to get device handle from index: %v", ret)
		}
		uuid, ret := dev.GetUUID()
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("failed to get device UUID: %v", ret)
		}
		return device.Identifier(uuid), nil
	}

	if id.IsMigIndex() {
		var gpuIdx, migIdx int
		var parent nvml.Device
		split := strings.SplitN(string(id), ":", 2)
		if gpuIdx, err = strconv.Atoi(split[0]); err != nil {
			return "", fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		if migIdx, err = strconv.Atoi(split[1]); err != nil {
			return "", fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		parent, ret := l.nvmllib.DeviceGetHandleByIndex(gpuIdx)
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("failed to get parent device handle: %v", ret)
		}
		mig, ret := parent.GetMigDeviceHandleByIndex(migIdx)
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("failed to get MIG handle by index: %v", ret)
		}
		uuid, ret := mig.GetUUID()
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("failed to get MIG UUID: %v", ret)
		}
		return device.Identifier(uuid), nil
	}

	return "", fmt.Errorf("identifier is not a valid UUID or index: %q", id)
}

func (l *nvmllib) init() error {
	if r := l.nvmllib.Init(); r != nvml.SUCCESS {
		return fmt.Errorf("failed to initialize NVML: %w", r)
	}

	if l.nvsandboxutilslib == nil {
		return nil
	}
	if r := l.nvsandboxutilslib.Init(l.driverRoot); r != nvsandboxutils.SUCCESS {
		l.logger.Warningf("Failed to init nvsandboxutils: %v; ignoring", r)
		l.nvsandboxutilslib = nil
	}
	return nil
}

func (l *nvmllib) tryShutdown() {
	if l.nvsandboxutilslib != nil {
		if r := l.nvsandboxutilslib.Shutdown(); r != nvsandboxutils.SUCCESS {
			l.logger.Warningf("failed to shutdown nvsandboxutils: %v", r)
		}
	}
	if r := l.nvmllib.Shutdown(); r != nvml.SUCCESS {
		l.logger.Warningf("failed to shutdown NVML: %v", r)
	}
}

type deviceSpecGeneratorsWithAndShutdown struct {
	*nvmllib
	DeviceSpecGenerator
}

func (l *nvmllib) withInit(dsg DeviceSpecGenerator) DeviceSpecGenerator {
	return &deviceSpecGeneratorsWithAndShutdown{
		nvmllib:             l,
		DeviceSpecGenerator: dsg,
	}
}

// GetDeviceSpecs ensures that the init and shutdown are called before (and
// after) generating the required device specs.
func (d *deviceSpecGeneratorsWithAndShutdown) GetDeviceSpecs() ([]specs.Device, error) {
	if err := d.init(); err != nil {
		return nil, err
	}
	defer d.tryShutdown()

	return d.DeviceSpecGenerator.GetDeviceSpecs()
}
