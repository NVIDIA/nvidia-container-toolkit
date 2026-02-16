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

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
)

type nvcdilib struct {
	logger logger.Interface
	platformlibs
	// TODO: This is currently only used for the gated discoverer and could
	// be captured there.
	mode         Mode
	deviceNamers DeviceNamers
	// TODO: We should use the devRoot associated with the driver.
	devRoot            string
	librarySearchPaths []string

	csv csvOptions

	driver *root.Driver

	featureFlags map[FeatureFlag]bool

	hookCreator  discover.HookCreator
	editsFactory edits.Factory
}

// New creates a new nvcdi library
func New(opts ...Option) (Interface, error) {
	o := populateOptions(opts...)

	l := &nvcdilib{
		logger:       o.logger,
		platformlibs: o.platformlibs,
		driver: root.New(
			o.getDriverOptions()...,
		),
		devRoot:      o.devRoot,
		deviceNamers: o.deviceNamers,

		librarySearchPaths: slices.Clone(o.librarySearchPaths),
		featureFlags:       o.featureFlags,
		mode:               o.mode,

		csv: o.csv,
	}

	var factory deviceSpecGeneratorFactory
	disabledHooks := slices.Clone(o.disabledHooks)
	vendor := o.vendor
	class := o.class
	switch l.mode {
	case ModeCSV:
		factory = l.asCSVLib()
	case ModeManagement:
		if vendor == "" {
			vendor = "management.nvidia.com"
		}
		// Management containers in general do not require CUDA Forward compatibility.
		disabledHooks = append(disabledHooks, HookEnableCudaCompat, DisableDeviceNodeModificationHook)
		factory = (*managementlib)(l)
	case ModeNvml:
		factory = (*nvmllib)(l)
	case ModeWsl:
		factory = (*wsllib)(l)
	case ModeGdrcopy, ModeGds, ModeMofed, ModeNvswitch:
		if class == "" {
			class = string(l.mode)
		}
		factory = (*gatedlib)(l)
	case ModeImex:
		if class == "" {
			class = classImexChannel
		}
		factory = (*imexlib)(l)
	default:
		return nil, fmt.Errorf("unknown mode %q", l.mode)
	}

	// Create a hook creator with the disabled hooks and the non-mode-dependent
	// options.
	l.hookCreator = discover.NewHookCreator(
		discover.WithNVIDIACDIHookPath(o.nvidiaCDIHookPath),
		discover.WithEnabledHooks(o.enabledHooks...),
		discover.WithLdconfigPath(o.ldconfigPath),
		discover.WithDisabledHooks(disabledHooks...),
	)

	l.editsFactory = edits.NewFactory(
		edits.WithLogger(o.logger),
		edits.WithNoAdditionalGIDsForDeviceNodes(o.featureFlags[FeatureNoAdditionalGIDsForDeviceNodes]),
	)

	w := wrapper{
		factory:             factory,
		vendor:              vendor,
		class:               class,
		mergedDeviceOptions: o.mergedDeviceOptions,
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

func (o *options) getNvmlLib() nvml.Interface {
	if o.nvmllib != nil {
		return o.nvmllib
	}

	var nvmlOpts []nvml.LibraryOption
	candidates, err := o.driverLibraryLocator().Locate("libnvidia-ml.so.1")
	if err != nil {
		o.logger.Warningf("Ignoring error in locating libnvidia-ml.so.1: %v", err)
	} else {
		libNvidiaMlPath := candidates[0]
		o.logger.Infof("Using %v", libNvidiaMlPath)
		nvmlOpts = append(nvmlOpts, nvml.WithLibraryPath(libNvidiaMlPath))
	}
	return nvml.New(nvmlOpts...)
}

// getNvsandboxUtilsLib returns the nvsandboxutilslib to use for CDI spec
// generation.
func (o *options) getNvsandboxUtilsLib() nvsandboxutils.Interface {
	if o.featureFlags[FeatureDisableNvsandboxUtils] {
		return nil
	}
	if o.nvsandboxutilslib != nil {
		return o.nvsandboxutilslib
	}

	var nvsandboxutilsOpts []nvsandboxutils.LibraryOption
	// Set the library path for libnvidia-sandboxutils
	candidates, err := o.driverLibraryLocator().Locate("libnvidia-sandboxutils.so.1")
	if err != nil {
		o.logger.Warningf("Ignoring error in locating libnvidia-sandboxutils.so.1: %v", err)
	} else {
		libNvidiaSandboxutilsPath := candidates[0]
		o.logger.Infof("Using %v", libNvidiaSandboxutilsPath)
		nvsandboxutilsOpts = append(nvsandboxutilsOpts, nvsandboxutils.WithLibraryPath(libNvidiaSandboxutilsPath))
	}

	// We try to initialize the library once to ensure that we have a valid installation.
	lib := nvsandboxutils.New(nvsandboxutilsOpts...)
	// TODO: Should this accept the driver root or the devRoot?
	if r := lib.Init(o.driverRoot); r != nvsandboxutils.SUCCESS {
		o.logger.Warningf("Failed to init nvsandboxutils: %v; ignoring", r)
		return nil
	}
	defer func() {
		_ = lib.Shutdown()
	}()

	return lib
}

func (o *options) getDriverOptions() []root.Option {
	return []root.Option{
		root.WithLogger(o.logger),
		root.WithDriverRoot(o.driverRoot),
		root.WithDevRoot(o.devRoot),
		root.WithLibrarySearchPaths(o.librarySearchPaths...),
		root.WithConfigSearchPaths(o.configSearchPaths...),
		root.WithVersioner(
			root.FirstOf(
				nvsandboxutilslibWithVersion(o.nvsandboxutilslib),
				nvmllibWithVersion(o.nvmllib),
			),
		),
	}
}
