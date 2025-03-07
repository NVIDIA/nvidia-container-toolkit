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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type nvcdilib struct {
	logger             logger.Interface
	nvmllib            nvml.Interface
	nvsandboxutilslib  nvsandboxutils.Interface
	mode               Mode
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

	disabledHooks disabledHooks
}

// New creates a new nvcdi library
func New(opts ...Option) (Interface, error) {
	l := &nvcdilib{
		disabledHooks: make(disabledHooks),
	}
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
		root.WithConfigSearchPaths(l.configSearchPaths...),
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
	// TODO: Repeated calls to nvsandboxutils.Init and Shutdown are causing
	// segmentation violations. Here we disabled nvsandbox utils unless explicitly
	// specified.
	// This will be reenabled as soon as we have more visibility into why this is
	// happening and a mechanism to detect and disable this if required.
	// if l.nvsandboxutilslib == nil {
	// 	var nvsandboxutilsOpts []nvsandboxutils.LibraryOption
	// 	// Set the library path for libnvidia-sandboxutils
	// 	candidates, err := l.driver.Libraries().Locate("libnvidia-sandboxutils.so.1")
	// 	if err != nil {
	// 		l.logger.Warningf("Ignoring error in locating libnvidia-sandboxutils.so.1: %v", err)
	// 	} else {
	// 		libNvidiaSandboxutilsPath := candidates[0]
	// 		l.logger.Infof("Using %v", libNvidiaSandboxutilsPath)
	// 		nvsandboxutilsOpts = append(nvsandboxutilsOpts, nvsandboxutils.WithLibraryPath(libNvidiaSandboxutilsPath))
	// 	}
	// 	l.nvsandboxutilslib = nvsandboxutils.New(nvsandboxutilsOpts...)
	// }
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
		// Management containers in general do not require CUDA Forward compatibility.
		l.disabledHooks[HookEnableCudaCompat] = true
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
	case ModeImex:
		if l.class == "" {
			l.class = classImexChannel
		}
		lib = (*imexlib)(l)
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

// getCudaVersion returns the CUDA version of the current system.
func (l *nvcdilib) getCudaVersion() (string, error) {
	version, err := l.getCudaVersionNvsandboxutils()
	if err == nil {
		return version, err
	}

	// Fallback to NVML
	return l.getCudaVersionNvml()
}

func (l *nvcdilib) getCudaVersionNvml() (string, error) {
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

func (l *nvcdilib) getCudaVersionNvsandboxutils() (string, error) {
	if l.nvsandboxutilslib == nil {
		return "", fmt.Errorf("libnvsandboxutils is not available")
	}

	// Sandboxutils initialization should happen before this function is called
	version, ret := l.nvsandboxutilslib.GetDriverVersion()
	if ret != nvsandboxutils.SUCCESS {
		return "", fmt.Errorf("%v", ret)
	}
	return version, nil
}
