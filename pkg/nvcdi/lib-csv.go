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
	"slices"
	"strconv"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/google/uuid"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra"
)

type csvlib nvcdilib

type mixedcsvlib nvcdilib

var _ deviceSpecGeneratorFactory = (*csvlib)(nil)

// DeviceSpecGenerators creates a set of generators for the specified set of
// devices.
// If NVML is not available or the disable-multiple-csv-devices feature flag is
// enabled, a single device is assumed.
func (l *csvlib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	if l.featureFlags[FeatureDisableMultipleCSVDevices] {
		return l.purecsvDeviceSpecGenerators(ids...)
	}
	hasNVML, _ := l.infolib.HasNvml()
	if !hasNVML {
		return l.purecsvDeviceSpecGenerators(ids...)
	}
	mixed, err := l.mixedDeviceSpecGenerators(ids...)
	if err != nil {
		l.logger.Warningf("Failed to create mixed CSV spec generator; falling back to pure CSV implementation: %v", err)
		return l.purecsvDeviceSpecGenerators(ids...)
	}
	return mixed, nil
}

func (l *csvlib) purecsvDeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	for _, id := range ids {
		switch id {
		case "all":
		case "0":
		default:
			return nil, fmt.Errorf("unsupported device id: %v", id)
		}
	}
	g := &csvDeviceGenerator{
		csvlib: l,
		index:  0,
		uuid:   "",
	}
	return g, nil
}

func (l *csvlib) mixedDeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	return (*mixedcsvlib)(l).DeviceSpecGenerators(ids...)
}

// A csvDeviceGenerator generates CDI specs for a device based on a set of
// platform-specific CSV files.
type csvDeviceGenerator struct {
	*csvlib
	index                 int
	uuid                  string
	onlyDeviceNodes       []string
	additionalDeviceNodes []string
}

func (l *csvDeviceGenerator) GetUUID() (string, error) {
	return l.uuid, nil
}

// GetDeviceSpecs returns the CDI device specs for a single device.
func (l *csvDeviceGenerator) GetDeviceSpecs() ([]specs.Device, error) {
	mountSpecs := tegra.MountSpecsFromCSVFiles(l.logger, l.csvFiles...)
	if len(l.onlyDeviceNodes) > 0 {
		mountSpecs = tegra.Merge(
			tegra.WithoutRegularDeviceNodes(mountSpecs),
			tegra.DeviceNodes(l.onlyDeviceNodes...),
		)
	}
	d, err := tegra.New(
		tegra.WithLogger(l.logger),
		tegra.WithDriverRoot(l.driverRoot),
		tegra.WithDevRoot(l.devRoot),
		tegra.WithHookCreator(l.hookCreator),
		tegra.WithLdconfigPath(l.ldconfigPath),
		tegra.WithLibrarySearchPaths(l.librarySearchPaths...),
		tegra.WithMountSpecsByPath(
			tegra.Filter(
				tegra.Merge(
					mountSpecs,
					tegra.DeviceNodes(l.additionalDeviceNodes...),
				),
				tegra.Merge(
					tegra.Symlinks(l.csvIgnorePatterns...),
				),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for CSV files: %v", err)
	}
	e, err := edits.FromDiscoverer(d)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for CSV files: %v", err)
	}

	names, err := l.deviceNamers.GetDeviceNames(l.index, l)
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

func (l *mixedcsvlib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	asNvmlLib := (*nvmllib)(l)
	err := asNvmlLib.init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nvml: %w", err)
	}
	defer asNvmlLib.tryShutdown()

	if slices.Contains(ids, "all") {
		ids, err = l.getAllDeviceIndices()
		if err != nil {
			return nil, fmt.Errorf("failed to get device indices: %w", err)
		}
	}

	var DeviceSpecGenerators DeviceSpecGenerators
	for _, id := range ids {
		generator, err := l.deviceSpecGeneratorForId(device.Identifier(id))
		if err != nil {
			return nil, fmt.Errorf("failed to create device spec generator for device %q: %w", id, err)
		}
		DeviceSpecGenerators = append(DeviceSpecGenerators, generator)
	}

	return DeviceSpecGenerators, nil
}

func (l *mixedcsvlib) getAllDeviceIndices() ([]string, error) {
	numDevices, ret := l.nvmllib.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("faled to get device count: %v", ret)
	}

	var allIndices []string
	for index := range numDevices {
		allIndices = append(allIndices, fmt.Sprintf("%d", index))
	}
	return allIndices, nil
}

func (l *mixedcsvlib) deviceSpecGeneratorForId(id device.Identifier) (DeviceSpecGenerator, error) {
	switch {
	case id.IsGpuUUID(), isIntegratedGPUID(id):
		uuid := string(id)
		device, ret := l.nvmllib.DeviceGetHandleByUUID(uuid)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device handle from UUID %q: %v", uuid, ret)
		}
		index, ret := device.GetIndex()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device index: %v", ret)
		}
		return l.csvDeviceSpecGenerator(index, uuid, device)
	case id.IsGpuIndex():
		index, err := strconv.Atoi(string(id))
		if err != nil {
			return nil, fmt.Errorf("failed to convert device index to an int: %w", err)
		}
		device, ret := l.nvmllib.DeviceGetHandleByIndex(index)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device handle from index: %v", ret)
		}
		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get UUID: %v", ret)
		}
		return l.csvDeviceSpecGenerator(index, uuid, device)
	case id.IsMigUUID():
		fallthrough
	case id.IsMigIndex():
		return nil, fmt.Errorf("generating a CDI spec for MIG id %q is not supported in CSV mode", id)
	}
	return nil, fmt.Errorf("identifier is not a valid UUID or index: %q", id)
}

func (l *mixedcsvlib) csvDeviceSpecGenerator(index int, uuid string, device nvml.Device) (DeviceSpecGenerator, error) {
	var additionalDeviceNodes []string
	isIntegrated, err := isIntegratedGPU(device)
	if err != nil {
		return nil, fmt.Errorf("is-integrated check failed for device (index=%v,uuid=%v)", index, uuid)
	}
	if !isIntegrated {
		additionalDeviceNodes = []string{
			"/dev/nvidia-uvm",
			"/dev/nvidia-uvm-tools",
		}
	}
	g := &csvDeviceGenerator{
		csvlib:                (*csvlib)(l),
		index:                 index,
		uuid:                  uuid,
		onlyDeviceNodes:       []string{fmt.Sprintf("/dev/nvidia%d", index)},
		additionalDeviceNodes: additionalDeviceNodes,
	}
	return g, nil
}

func isIntegratedGPUID(id device.Identifier) bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

// isIntegratedGPU checks whether the specified device is an integrated GPU.
// As a proxy we check the PCI Bus if for thes
// TODO: This should be replaced by an explicit NVML call once available.
func isIntegratedGPU(d nvml.Device) (bool, error) {
	pciInfo, ret := d.GetPciInfo()
	if ret == nvml.ERROR_NOT_SUPPORTED {
		name, ret := d.GetName()
		if ret != nvml.SUCCESS {
			return false, fmt.Errorf("failed to get device name: %v", ret)
		}
		return isIntegratedGPUName(name), nil
	}
	if ret != nvml.SUCCESS {
		return false, fmt.Errorf("failed to get PCI info: %v", ret)
	}

	if pciInfo.Domain != 0 {
		return false, nil
	}
	if pciInfo.Bus != 1 {
		return false, nil
	}
	return pciInfo.Device == 0, nil
}

// isIntegratedGPUName returns true if the specified device name is associated
// with a known iGPU.
//
// TODO: Consider making go-nvlib/pkg/nvlib/info/isIntegratedGPUName public
// instead.
func isIntegratedGPUName(name string) bool {
	if strings.Contains(name, "(nvgpu)") {
		return true
	}
	if strings.Contains(name, "NVIDIA Thor") {
		return true
	}
	return false
}
