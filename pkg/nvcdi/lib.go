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

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type wrapper struct {
	Interface

	vendor string
	class  string

	mergedDeviceOptions []transform.MergedDeviceOption
}

type nvcdilib struct {
	logger             logger.Interface
	nvmllib            nvml.Interface
	mode               string
	devicelib          device.Interface
	deviceNamers       DeviceNamers
	driverRoot         string
	devRoot            string
	nvidiaCDIHookPath  string
	ldconfigPath       string
	configSearchPaths  []string
	librarySearchPaths []string

	csvFiles          []string
	csvIgnorePatterns []string

	vendor string
	class  string

	driver  *root.Driver
	infolib info.Interface

	mergedDeviceOptions []transform.MergedDeviceOption
}

// New creates a new nvcdi library
func New(opts ...Option) (Interface, error) {
	l := &nvcdilib{}
	for _, opt := range opts {
		opt(l)
	}
	if l.mode == "" {
		l.mode = ModeAuto
	}
	if l.logger == nil {
		l.logger = logger.New()
	}
	if len(l.deviceNamers) == 0 {
		indexNamer, _ := NewDeviceNamer(DeviceNameStrategyIndex)
		l.deviceNamers = []DeviceNamer{indexNamer}
	}
	if l.nvidiaCDIHookPath == "" {
		l.nvidiaCDIHookPath = "/usr/bin/nvidia-cdi-hook"
	}
	if l.driverRoot == "" {
		l.driverRoot = "/"
	}
	if l.devRoot == "" {
		l.devRoot = l.driverRoot
	}
	l.driver = root.New(
		root.WithLogger(l.logger),
		root.WithDriverRoot(l.driverRoot),
		root.WithLibrarySearchPaths(l.librarySearchPaths...),
	)
	if l.nvmllib == nil {
		var nvmlOpts []nvml.LibraryOption
		candidates, err := l.driver.Libraries().Locate("libnvidia-ml.so.1")
		if err != nil {
			l.logger.Warningf("Ignoring error in locating libnvidia-ml.so.1: %v", err)
		} else {
			libNvidiaMlPath := candidates[0]
			l.logger.Infof("Using %v", libNvidiaMlPath)
			nvmlOpts = append(nvmlOpts, nvml.WithLibraryPath(libNvidiaMlPath))
		}
		l.nvmllib = nvml.New(nvmlOpts...)
	}
	if l.devicelib == nil {
		l.devicelib = device.New(l.nvmllib)
	}
	if l.infolib == nil {
		l.infolib = info.New(
			info.WithRoot(l.driverRoot),
			info.WithLogger(l.logger),
			info.WithNvmlLib(l.nvmllib),
			info.WithDeviceLib(l.devicelib),
		)
	}

	var lib Interface
	switch l.resolveMode() {
	case ModeCSV:
		if len(l.csvFiles) == 0 {
			l.csvFiles = csv.DefaultFileList()
		}
		lib = (*csvlib)(l)
	case ModeManagement:
		if l.vendor == "" {
			l.vendor = "management.nvidia.com"
		}
		lib = (*managementlib)(l)
	case ModeNvml:
		lib = (*nvmllib)(l)
	case ModeWsl:
		lib = (*wsllib)(l)
	case ModeGds:
		if l.class == "" {
			l.class = "gds"
		}
		lib = (*gdslib)(l)
	case ModeMofed:
		if l.class == "" {
			l.class = "mofed"
		}
		lib = (*mofedlib)(l)
	default:
		return nil, fmt.Errorf("unknown mode %q", l.mode)
	}

	w := wrapper{
		Interface:           lib,
		vendor:              l.vendor,
		class:               l.class,
		mergedDeviceOptions: l.mergedDeviceOptions,
	}
	return &w, nil
}

// GetSpec combines the device specs and common edits from the wrapped Interface to a single spec.Interface.
func (l *wrapper) GetSpec() (spec.Interface, error) {
	deviceSpecs, err := l.GetAllDeviceSpecs()
	if err != nil {
		return nil, err
	}

	edits, err := l.GetCommonEdits()
	if err != nil {
		return nil, err
	}

	return spec.New(
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*edits.ContainerEdits),
		spec.WithVendor(l.vendor),
		spec.WithClass(l.class),
		spec.WithMergedDeviceOptions(l.mergedDeviceOptions...),
	)
}

// GetCommonEdits returns the wrapped edits and adds additional edits on top.
func (m *wrapper) GetCommonEdits() (*cdi.ContainerEdits, error) {
	edits, err := m.Interface.GetCommonEdits()
	if err != nil {
		return nil, err
	}
	edits.Env = append(edits.Env, "NVIDIA_VISIBLE_DEVICES=void")

	return edits, nil
}

// resolveMode resolves the mode for CDI spec generation based on the current system.
func (l *nvcdilib) resolveMode() (rmode string) {
	if l.mode != ModeAuto {
		return l.mode
	}
	defer func() {
		l.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	platform := l.infolib.ResolvePlatform()
	switch platform {
	case info.PlatformNVML:
		return ModeNvml
	case info.PlatformTegra:
		return ModeCSV
	case info.PlatformWSL:
		return ModeWsl
	}
	l.logger.Warningf("Unsupported platform detected: %v; assuming %v", platform, ModeNvml)
	return ModeNvml
}

// getCudaVersion returns the CUDA version of the current system.
func (l *nvcdilib) getCudaVersion() (string, error) {
	if hasNVML, reason := l.infolib.HasNvml(); !hasNVML {
		return "", fmt.Errorf("nvml not detected: %v", reason)
	}
	if l.nvmllib == nil {
		return "", fmt.Errorf("nvml library not initialized")
	}
	r := l.nvmllib.Init()
	if r != nvml.SUCCESS {
		return "", fmt.Errorf("failed to initialize nvml: %v", r)
	}
	defer func() {
		if r := l.nvmllib.Shutdown(); r != nvml.SUCCESS {
			l.logger.Warningf("failed to shutdown NVML: %v", r)
		}
	}()

	version, r := l.nvmllib.SystemGetDriverVersion()
	if r != nvml.SUCCESS {
		return "", fmt.Errorf("failed to get driver version: %v", r)
	}
	return version, nil
}
