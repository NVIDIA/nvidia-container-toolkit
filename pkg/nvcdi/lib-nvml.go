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

var _ wrapped = (*nvmllib)(nil)

// GetCommonEdits generates a CDI specification that can be used for ANY devices
func (l *nvmllib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	common, err := l.newCommonNVMLDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for common entities: %v", err)
	}

	return edits.FromDiscoverer(common)
}

// GetDeviceSpecsByID returns the CDI device specs for the devices represented
// by the requested identifiers. Here an identifier is one of the following:
// * an index of a GPU or MIG device
// * a UUID of a GPU or MIG device
func (l *nvmllib) GetDeviceSpecsByID(ids ...string) ([]specs.Device, error) {
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

	generators, err := l.getDeviceSpecGeneratorsForIDs(ids...)
	if err != nil {
		return nil, err
	}

	return generators.GetDeviceSpecs()
}

func (l *nvmllib) newDeviceSpecGeneratorFromNVMLDevice(id string, nvmlDevice nvml.Device) (deviceSpecGenerator, error) {
	isMig, ret := nvmlDevice.IsMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return nil, ret
	}
	if isMig {
		return l.newMIGDeviceSpecGeneratorFromNVMLDevice(id, nvmlDevice)
	}

	return l.newFullGPUDeviceSpecGeneratorFromNVMLDevice(id, nvmlDevice)
}

func (l *nvmllib) getDeviceSpecGeneratorsForIDs(ids ...string) (deviceSpecGenerators, error) {
	var identifiers []device.Identifier
	for _, id := range ids {
		if id == "all" {
			return l.getDeviceSpecGeneratorsForAllDevices()
		}
		identifiers = append(identifiers, device.Identifier(id))
	}

	devices, err := l.getNVMLDevicesByID(identifiers...)
	if err != nil {
		return nil, err
	}

	var DeviceSpecGenerators deviceSpecGenerators
	for i, device := range devices {
		editor, err := l.newDeviceSpecGeneratorFromNVMLDevice(ids[i], device)
		if err != nil {
			return nil, err
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, editor)
	}

	return DeviceSpecGenerators, nil
}

func (l *nvmllib) getDeviceSpecGeneratorsForAllDevices() ([]deviceSpecGenerator, error) {
	var DeviceSpecGenerators []deviceSpecGenerator
	err := l.devicelib.VisitDevices(func(i int, d device.Device) error {
		e := &fullGPUDeviceSpecGenerator{
			nvmllib: l,
			id:      fmt.Sprintf("%d", i),
			device:  d,
		}

		DeviceSpecGenerators = append(DeviceSpecGenerators, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get full GPU device editors: %w", err)
	}

	err = l.devicelib.VisitMigDevices(func(i int, d device.Device, j int, mig device.MigDevice) error {
		e := &migDeviceSpecGenerator{
			nvmllib: l,
			id:      fmt.Sprintf("%d:%d", i, j),
			parent:  d,
			device:  mig,
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get MIG device editors: %w", err)
	}

	return DeviceSpecGenerators, nil
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

type deviceSpecGenerators []deviceSpecGenerator

// GetDeviceSpecs returns the combined specs for each device spec generator.
func (g deviceSpecGenerators) GetDeviceSpecs() ([]specs.Device, error) {
	var allDeviceSpecs []specs.Device
	for _, dsg := range g {
		if dsg == nil {
			continue
		}
		deviceSpecs, err := dsg.GetDeviceSpecs()
		if err != nil {
			return nil, err
		}
		allDeviceSpecs = append(allDeviceSpecs, deviceSpecs...)
	}

	return allDeviceSpecs, nil
}
