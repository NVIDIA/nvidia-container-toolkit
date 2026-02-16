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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type options struct {
	logger logger.Interface
	platformlibs
	mode               Mode
	deviceNamers       DeviceNamers
	driverRoot         string
	devRoot            string
	nvidiaCDIHookPath  string
	ldconfigPath       string
	configSearchPaths  []string
	librarySearchPaths []string

	csv csvOptions

	vendor string
	class  string

	mergedDeviceOptions []transform.MergedDeviceOption

	featureFlags map[FeatureFlag]bool

	disabledHooks []discover.HookName
	enabledHooks  []discover.HookName
}

type platformlibs struct {
	nvmllib   nvml.Interface
	devicelib device.Interface
	infolib   info.Interface

	nvsandboxutilslib nvsandboxutils.Interface
}

// populateOptions applies the functional options and resolves the required
// defaults.
func populateOptions(opts ...Option) *options {
	o := &options{
		mode:              ModeAuto,
		driverRoot:        "/",
		nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = logger.New()
	}
	if len(o.deviceNamers) == 0 {
		indexNamer, _ := NewDeviceNamer(DeviceNameStrategyIndex)
		o.deviceNamers = []DeviceNamer{indexNamer}
	}
	if o.devRoot == "" {
		o.devRoot = o.driverRoot
	}
	if o.nvsandboxutilslib == nil {
		o.nvsandboxutilslib = o.getNvsandboxUtilsLib()
	}
	if o.nvmllib == nil {
		o.nvmllib = o.getNvmlLib()
	}
	if o.devicelib == nil {
		o.devicelib = device.New(o.nvmllib)
	}
	if o.infolib == nil {
		o.infolib = info.New(
			info.WithRoot(o.driverRoot),
			info.WithLogger(o.logger),
			info.WithNvmlLib(o.nvmllib),
			info.WithDeviceLib(o.devicelib),
		)
	}
	o.mode = o.resolveMode()

	return o
}

func (o *options) driverLibraryLocator() lookup.Locator {
	return lookup.NewLibraryLocator(
		lookup.WithLogger(o.logger),
		lookup.WithRoot(o.driverRoot),
		lookup.WithSearchPaths(o.librarySearchPaths...),
	)
}

// Option is a function that configures the nvcdi library options.
type Option func(*options)

// WithDeviceLib sets the device library for the library
func WithDeviceLib(devicelib device.Interface) Option {
	return func(l *options) {
		l.devicelib = devicelib
	}
}

// WithInfoLib sets the info library for CDI spec generation.
func WithInfoLib(infolib info.Interface) Option {
	return func(l *options) {
		l.infolib = infolib
	}
}

// WithDeviceNamers sets the device namer for the library
func WithDeviceNamers(namers ...DeviceNamer) Option {
	return func(l *options) {
		l.deviceNamers = namers
	}
}

// WithDriverRoot sets the driver root for the library
func WithDriverRoot(root string) Option {
	return func(l *options) {
		l.driverRoot = root
	}
}

// WithDevRoot sets the root where /dev is located.
func WithDevRoot(root string) Option {
	return func(l *options) {
		l.devRoot = root
	}
}

// WithLogger sets the logger for the library
func WithLogger(logger logger.Interface) Option {
	return func(l *options) {
		l.logger = logger
	}
}

// WithNVIDIACTKPath sets the path to the NVIDIA Container Toolkit CLI path for the library
//
// Deprecated: Use WithNVIDIACDIHookPath instead.
func WithNVIDIACTKPath(path string) Option {
	return WithNVIDIACDIHookPath(path)
}

// WithNVIDIACDIHookPath sets the path to the NVIDIA Container Toolkit CLI path for the library
func WithNVIDIACDIHookPath(path string) Option {
	return func(l *options) {
		l.nvidiaCDIHookPath = path
	}
}

// WithLdconfigPath sets the path to the ldconfig program
func WithLdconfigPath(path string) Option {
	return func(l *options) {
		l.ldconfigPath = path
	}
}

// WithNvmlLib sets the nvml library for the library
func WithNvmlLib(nvmllib nvml.Interface) Option {
	return func(l *options) {
		l.nvmllib = nvmllib
	}
}

// WithMode sets the discovery mode for the library
func WithMode[m modeConstraint](mode m) Option {
	return func(l *options) {
		l.mode = Mode(mode)
	}
}

// WithVendor sets the vendor for the library
func WithVendor(vendor string) Option {
	return func(o *options) {
		o.vendor = vendor
	}
}

// WithClass sets the class for the library
func WithClass(class string) Option {
	return func(o *options) {
		o.class = class
	}
}

// WithMergedDeviceOptions sets the merged device options for the library
// If these are not set, no merged device will be generated.
func WithMergedDeviceOptions(opts ...transform.MergedDeviceOption) Option {
	return func(o *options) {
		o.mergedDeviceOptions = opts
	}
}

// WithCSVFiles sets the CSV files for the library
func WithCSVFiles(csvFiles []string) Option {
	return func(o *options) {
		o.csv.Files = csvFiles
	}
}

// WithCSVIgnorePatterns sets the ignore patterns for entries in the CSV files.
func WithCSVIgnorePatterns(csvIgnorePatterns []string) Option {
	return func(o *options) {
		o.csv.IgnorePatterns = csvIgnorePatterns
	}
}

// WithCSVCompatContainerRoot sets the compat root to use for the container in
// the case of nvgpu-only devices.
func WithCSVCompatContainerRoot(csvCompatContainerRoot string) Option {
	return func(o *options) {
		o.csv.CompatContainerRoot = csvCompatContainerRoot
	}
}

// WithConfigSearchPaths sets the search paths for config files.
func WithConfigSearchPaths(paths []string) Option {
	return func(o *options) {
		o.configSearchPaths = paths
	}
}

// WithLibrarySearchPaths sets the library search paths.
// This is currently only used for CSV-mode.
func WithLibrarySearchPaths(paths []string) Option {
	return func(o *options) {
		o.librarySearchPaths = paths
	}
}

// WithDisabledHooks allows specific hooks to be disabled.
func WithDisabledHooks[T string | HookName](hooks ...T) Option {
	return func(o *options) {
		for _, hook := range hooks {
			o.disabledHooks = append(o.disabledHooks, discover.HookName(hook))
		}
	}
}

// WithEnabledHooks explicitly enables a specific set of hooks.
// If a hook is explicitly enabled, this takes precedence over it being disabled.
func WithEnabledHooks[T string | HookName](hooks ...T) Option {
	return func(o *options) {
		for _, hook := range hooks {
			o.enabledHooks = append(o.enabledHooks, discover.HookName(hook))
		}
	}
}

// WithFeatureFlags allows the specified set of features to be toggled on.
func WithFeatureFlags[T string | FeatureFlag](featureFlags ...T) Option {
	return func(o *options) {
		if o.featureFlags == nil {
			o.featureFlags = make(map[FeatureFlag]bool)
		}
		for _, featureFlag := range featureFlags {
			// The initial release of the FeatureDisableNvsandboxUtils feature
			// flag included a typo which we handle here.
			if string(featureFlag) == "disable-nvsandbox-utils" {
				featureFlag = T(FeatureDisableNvsandboxUtils)
			}
			o.featureFlags[FeatureFlag(featureFlag)] = true
		}
	}
}

// WithDisabledHook allows specific hooks to be disabled.
// This option can be specified multiple times for each hook.
//
// Deprecated: Use WithDisabledHooks instead
func WithDisabledHook[T string | HookName](hook T) Option {
	return WithDisabledHooks(hook)
}

// WithFeatureFlag allows specified features to be toggled on.
// This option can be specified multiple times for each feature flag.
//
// Deprecated: Use WithFeatureFlags
func WithFeatureFlag[T string | FeatureFlag](featureFlag T) Option {
	return WithFeatureFlags(featureFlag)
}
