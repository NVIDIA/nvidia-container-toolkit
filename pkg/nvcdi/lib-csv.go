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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/google/uuid"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

const (
	defaultOrinCompatContainerRoot = "/usr/local/cuda/compat-orin"
)

type csvOptions struct {
	Files               []string
	IgnorePatterns      []string
	CompatContainerRoot string
}

type csvlib nvcdilib
type mixedcsvlib nvcdilib

var _ deviceSpecGeneratorFactory = (*csvlib)(nil)

// asCSVLib sets any CSV-specific defaults and casts the nvcdilib instance as a
// *csvlib.
func (l *nvcdilib) asCSVLib() *csvlib {
	if len(l.csv.Files) == 0 {
		l.csv.Files = csv.DefaultFileList()
	}
	if l.csv.CompatContainerRoot == "" {
		l.csv.CompatContainerRoot = defaultOrinCompatContainerRoot
	}
	return (*csvlib)(l)
}

// DeviceSpecGenerators creates a set of generators for the specified set of
// devices.
// If NVML is not available or the disable-multiple-csv-devices feature flag is
// enabled, a single device is assumed.
func (l *csvlib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	if l.usePureCSVDeviceSpecGenerator() {
		return l.purecsvDeviceSpecGenerators(ids...)
	}
	mixed, err := l.mixedDeviceSpecGenerators(ids...)
	if err != nil {
		l.logger.Warningf("Failed to create mixed CSV spec generator; falling back to pure CSV implementation: %v", err)
		return l.purecsvDeviceSpecGenerators(ids...)
	}
	return mixed, nil
}

func (l *csvlib) usePureCSVDeviceSpecGenerator() bool {
	if l.featureFlags[FeatureDisableMultipleCSVDevices] {
		return true
	}
	hasNVML, _ := l.infolib.HasNvml()
	if !hasNVML {
		return true
	}
	asNvmlLib := (*nvmllib)(l)
	err := asNvmlLib.init()
	if err != nil {
		return true
	}
	defer asNvmlLib.tryShutdown()

	numDevices, ret := l.nvmllib.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return true
	}

	return numDevices <= 1
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
	index int
	uuid  string
	mode  csvGeneratorMode
}

type csvGeneratorMode string

const (
	iGPUGeneratorMode = csvGeneratorMode("igpu")
	dGPUGeneratorMode = csvGeneratorMode("dgpu")
)

func (l *csvDeviceGenerator) GetUUID() (string, error) {
	return l.uuid, nil
}

// GetDeviceSpecs returns the CDI device specs for a single device.
func (l *csvDeviceGenerator) GetDeviceSpecs() ([]specs.Device, error) {
	deviceNodeDiscoverer, err := l.deviceNodeDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for device nodes from CSV files: %w", err)
	}
	e, err := edits.FromDiscoverer(deviceNodeDiscoverer)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for CSV files: %w", err)
	}

	names, err := l.deviceNamers.GetDeviceNames(l.index, l)
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

// deviceNodeDiscoverer creates a discoverer for the device nodes associated
// with the specified device.
// The CSV mount specs are used as the source for which device nodes are
// required with the following additions:
//
//   - Any regular device nodes (i.e. /dev/nvidia[0-9]+) are removed from the
//     input set.
//   - The device node (i.e. /dev/nvidia{{ .index }}) associated with this
//     particular device is added to the set of device nodes to be discovered.
func (l *csvDeviceGenerator) deviceNodeDiscoverer() (discover.Discover, error) {
	return tegra.New(
		tegra.WithLogger(l.logger),
		tegra.WithDriverRoot(l.driverRoot),
		tegra.WithDevRoot(l.devRoot),
		tegra.WithHookCreator(l.hookCreator),
		tegra.WithLdconfigPath(l.ldconfigPath),
		tegra.WithLibrarySearchPaths(l.librarySearchPaths...),
		tegra.WithMountSpecs(l.deviceNodeMountSpecs()),
	)
}

func (l *csvDeviceGenerator) deviceNodeMountSpecs() tegra.MountSpecPathsByTyper {
	mountSpecs := tegra.Transform(
		tegra.MountSpecsFromCSVFiles(l.logger, l.csv.Files...),
		// We remove non-device nodes.
		tegra.OnlyDeviceNodes(),
	)
	switch l.mode {
	case dGPUGeneratorMode:
		return tegra.Transform(
			mountSpecs,
			// For a dGPU we remove all regular device nodes (nvidia[0-9]+)
			// from the list of device nodes taken from the CSV mount specs.
			// The device nodes for the GPU are discovered for the full GPU.
			tegra.WithoutRegularDeviceNodes(),
			// We also ignore control device nodes since these are included in
			// the full GPU spec generator.
			tegra.Without(
				tegra.DeviceNodes(
					"/dev/nvidia-modeset",
					"/dev/nvidia-uvm-tools",
					"/dev/nvidia-uvm",
					"/dev/nvidiactl",
				),
			),
		)
	case iGPUGeneratorMode:
		return tegra.Merge(
			tegra.Transform(
				mountSpecs,
				// We remove the /dev/nvidia1 device node.
				// TODO: This assumes that the dGPU has the index 1 and remove
				// it from the set of device nodes.
				tegra.Without(tegra.DeviceNodes("/dev/nvidia1")),
			),
			// We add the display device from the iGPU.
			tegra.DeviceNodes("/dev/nvidia2"),
		)
	default:
		return mountSpecs
	}
}

// GetCommonEdits generates a CDI specification that can be used for ANY devices
// These explicitly do not include any device nodes.
func (l *csvlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	driverDiscoverer, err := l.driverDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create driver discoverer from CSV files: %w", err)
	}
	return edits.FromDiscoverer(driverDiscoverer)
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
	isIntegrated, err := isIntegratedGPU(device)
	if err != nil {
		return nil, fmt.Errorf("is-integrated check failed for device (index=%v,uuid=%v)", index, uuid)
	}

	if isIntegrated {
		return l.iGPUDeviceSpecGenerator(index, uuid)
	}

	return l.dGPUDeviceSpecGenerator(index, uuid)
}

func (l *mixedcsvlib) dGPUDeviceSpecGenerator(index int, uuid string) (DeviceSpecGenerator, error) {
	if index != 1 {
		return nil, fmt.Errorf("unexpected device index for dGPU: %d", index)
	}
	g := &csvDeviceGenerator{
		csvlib: (*csvlib)(l),
		index:  index,
		uuid:   uuid,
		mode:   dGPUGeneratorMode,
	}

	csvDeviceNodeDiscoverer, err := g.deviceNodeDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for devices nodes: %w", err)
	}

	// If this is not an integrated GPU, we also create a spec generator for
	// the full GPU.
	dgpu := (*nvmllib)(l).withInit(&fullGPUDeviceSpecGenerator{
		nvmllib: (*nvmllib)(l),
		uuid:    uuid,
		index:   index,
		// For the CSV case, we include the control device nodes at a
		// device level.
		additionalDiscoverers: []discover.Discover{
			(*nvmllib)(l).controlDeviceNodeDiscoverer(),
			csvDeviceNodeDiscoverer,
		},
		featureFlags: l.featureFlags,
	})
	return dgpu, nil
}

func (l *mixedcsvlib) iGPUDeviceSpecGenerator(index int, uuid string) (DeviceSpecGenerator, error) {
	if index != 0 {
		return nil, fmt.Errorf("unexpected device index for iGPU: %d", index)
	}
	g := &csvDeviceGenerator{
		csvlib: (*csvlib)(l),
		index:  index,
		uuid:   uuid,
		mode:   iGPUGeneratorMode,
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
		return info.IsIntegratedGPUName(name), nil
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

func (l *csvlib) driverDiscoverer() (discover.Discover, error) {
	mountSpecs := tegra.Transform(
		tegra.Transform(
			tegra.MountSpecsFromCSVFiles(l.logger, l.csv.Files...),
			tegra.WithoutDeviceNodes(),
		),
		tegra.IgnoreSymlinkMountSpecsByPattern(l.csv.IgnorePatterns...),
	)
	driverDiscoverer, err := tegra.New(
		tegra.WithLogger(l.logger),
		tegra.WithDriverRoot(l.driverRoot),
		tegra.WithDevRoot(l.devRoot),
		tegra.WithHookCreator(l.hookCreator),
		tegra.WithLdconfigPath(l.ldconfigPath),
		tegra.WithLibrarySearchPaths(l.librarySearchPaths...),
		tegra.WithMountSpecs(mountSpecs),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer from CSV files: %w", err)
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

// cudaCompatDiscoverer returns a discoverer for the CUDA forward compat hook
// on Tegra-based systems.
// If the system has NVML available, this is used to determine the driver
// version to be passed to the hook.
// On Orin-based systems, the compat library root in the container is also set.
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
		return nil
	}

	var cudaCompatContainerRoot string
	for _, name := range names {
		// TODO: Should this be overridable through a feature flag / config option?
		if strings.Contains(name, "Orin (nvgpu)") {
			// TODO: This should probably be a constant or configurable.
			cudaCompatContainerRoot = l.csv.CompatContainerRoot
			break
		}
	}

	return discover.NewCUDACompatHookDiscoverer(l.logger, l.hookCreator, version, cudaCompatContainerRoot)
}
