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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
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

	featureFlags map[FeatureFlag]bool

	disabledHooks []discover.HookName
	enabledHooks  []discover.HookName
	hookCreator   discover.HookCreator

	editsFactory edits.Factory
}

// New creates a new nvcdi library
func New(opts ...Option) (Interface, error) {
	l := &nvcdilib{
		featureFlags: make(map[FeatureFlag]bool),
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
	if l.editsFactory == nil {
		l.editsFactory = edits.NewFactory(
			edits.WithLogger(l.logger),
			edits.WithNoAdditionalGIDsForDeviceNodes(l.featureFlags[FeatureNoAdditionalGIDsForDeviceNodes]),
		)
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

	l.nvmllib = l.getNvmlLib()
	l.nvsandboxutilslib = l.getNvsandboxUtilsLib()
	l.driver = l.getDriver(
		root.WithVersioner(
			root.FirstOf(
				nvsandboxutilslibWithVersion(l.nvsandboxutilslib),
				nvmllibWithVersion(l.nvmllib),
			),
		),
	)

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

	var factory deviceSpecGeneratorFactory
	switch l.resolveMode() {
	case ModeCSV:
		if len(l.csvFiles) == 0 {
			l.csvFiles = csv.DefaultFileList()
		}
		factory = (*csvlib)(l)
	case ModeManagement:
		if l.vendor == "" {
			l.vendor = "management.nvidia.com"
		}
		// Management containers in general do not require CUDA Forward compatibility.
		l.disabledHooks = append(l.disabledHooks, HookEnableCudaCompat, DisableDeviceNodeModificationHook)
		factory = (*managementlib)(l)
	case ModeNvml:
		factory = (*nvmllib)(l)
	case ModeWsl:
		factory = (*wsllib)(l)
	case ModeGdrcopy, ModeGds, ModeMofed:
		if l.class == "" {
			l.class = string(l.mode)
		}
		factory = (*gatedlib)(l)
	case ModeImex:
		if l.class == "" {
			l.class = classImexChannel
		}
		factory = (*imexlib)(l)
	default:
		return nil, fmt.Errorf("unknown mode %q", l.mode)
	}

	// create hookCreator
	l.hookCreator = discover.NewHookCreator(
		discover.WithNVIDIACDIHookPath(l.nvidiaCDIHookPath),
		discover.WithDisabledHooks(l.disabledHooks...),
		discover.WithEnabledHooks(l.enabledHooks...),
	)

	w := wrapper{
		factory:             factory,
		vendor:              l.vendor,
		class:               l.class,
		mergedDeviceOptions: l.mergedDeviceOptions,
	}
	return &w, nil
}

type nvmllibAsVersioner struct {
	nvml.Interface
}

func nvmllibWithVersion(nvmllib nvml.Interface) *nvmllibAsVersioner {
	if nvmllib == nil {
		return nil
	}
	return &nvmllibAsVersioner{
		Interface: nvmllib,
	}
}

func (l *nvmllibAsVersioner) Version() (string, error) {
	if l == nil || l.Interface == nil {
		return "", fmt.Errorf("nvml library not initialized")
	}

	r := l.Init()
	if r != nvml.SUCCESS {
		return "", fmt.Errorf("failed to initialize nvml: %v", r)
	}
	defer func() {
		_ = l.Shutdown()
	}()

	version, r := l.SystemGetDriverVersion()
	if r != nvml.SUCCESS {
		return "", fmt.Errorf("failed to get driver version: %v", r)
	}
	return version, nil
}

type nvsandboxutilslibAsVersioner struct {
	nvsandboxutils.Interface
}

func nvsandboxutilslibWithVersion(nvsandboxutilslib nvsandboxutils.Interface) *nvsandboxutilslibAsVersioner {
	if nvsandboxutilslib == nil {
		return nil
	}
	return &nvsandboxutilslibAsVersioner{
		Interface: nvsandboxutilslib,
	}
}

func (l *nvsandboxutilslibAsVersioner) Version() (string, error) {
	if l == nil || l.Interface == nil {
		return "", fmt.Errorf("libnvsandboxutils is not available")
	}

	// Sandboxutils initialization should happen before this function is called
	version, ret := l.GetDriverVersion()
	if ret != nvsandboxutils.SUCCESS {
		return "", fmt.Errorf("%v", ret)
	}
	return version, nil
}

func (l *nvcdilib) getNvmlLib() nvml.Interface {
	if l.nvmllib != nil {
		return l.nvmllib
	}

	var nvmlOpts []nvml.LibraryOption
	candidates, err := l.getDriver().Libraries().Locate("libnvidia-ml.so.1")
	if err != nil {
		l.logger.Warningf("Ignoring error in locating libnvidia-ml.so.1: %v", err)
	} else {
		libNvidiaMlPath := candidates[0]
		l.logger.Infof("Using %v", libNvidiaMlPath)
		nvmlOpts = append(nvmlOpts, nvml.WithLibraryPath(libNvidiaMlPath))
	}
	return nvml.New(nvmlOpts...)
}

// getNvsandboxUtilsLib returns the nvsandboxutilslib to use for CDI spec
// generation.
func (l *nvcdilib) getNvsandboxUtilsLib() nvsandboxutils.Interface {
	if l.featureFlags[FeatureDisableNvsandboxUtils] {
		return nil
	}
	if l.nvsandboxutilslib != nil {
		return l.nvsandboxutilslib
	}

	var nvsandboxutilsOpts []nvsandboxutils.LibraryOption
	// Set the library path for libnvidia-sandboxutils
	candidates, err := l.getDriver().Libraries().Locate("libnvidia-sandboxutils.so.1")
	if err != nil {
		l.logger.Warningf("Ignoring error in locating libnvidia-sandboxutils.so.1: %v", err)
	} else {
		libNvidiaSandboxutilsPath := candidates[0]
		l.logger.Infof("Using %v", libNvidiaSandboxutilsPath)
		nvsandboxutilsOpts = append(nvsandboxutilsOpts, nvsandboxutils.WithLibraryPath(libNvidiaSandboxutilsPath))
	}

	// We try to initialize the library once to ensure that we have a valid installation.
	lib := nvsandboxutils.New(nvsandboxutilsOpts...)
	if r := lib.Init(l.driverRoot); r != nvsandboxutils.SUCCESS {
		l.logger.Warningf("Failed to init nvsandboxutils: %v; ignoring", r)
		return nil
	}
	defer func() {
		_ = lib.Shutdown()
	}()

	return lib
}

func (l *nvcdilib) getDriver(additionalOptions ...root.Option) *root.Driver {
	options := []root.Option{
		root.WithLogger(l.logger),
		root.WithDriverRoot(l.driverRoot),
		root.WithLibrarySearchPaths(l.librarySearchPaths...),
		root.WithConfigSearchPaths(l.configSearchPaths...),
	}

	options = append(options, additionalOptions...)

	return root.New(options...)
}
