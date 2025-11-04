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
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

// Option is a function that configures the nvcdilib
type Option func(*nvcdilib)

// WithDeviceLib sets the device library for the library
func WithDeviceLib(devicelib device.Interface) Option {
	return func(l *nvcdilib) {
		l.devicelib = devicelib
	}
}

// WithInfoLib sets the info library for CDI spec generation.
func WithInfoLib(infolib info.Interface) Option {
	return func(l *nvcdilib) {
		l.infolib = infolib
	}
}

// WithDeviceNamers sets the device namer for the library
func WithDeviceNamers(namers ...DeviceNamer) Option {
	return func(l *nvcdilib) {
		l.deviceNamers = namers
	}
}

// WithDriverRoot sets the driver root for the library
func WithDriverRoot(root string) Option {
	return func(l *nvcdilib) {
		l.driverRoot = root
	}
}

// WithDevRoot sets the root where /dev is located.
func WithDevRoot(root string) Option {
	return func(l *nvcdilib) {
		l.devRoot = root
	}
}

// WithLogger sets the logger for the library
func WithLogger(logger logger.Interface) Option {
	return func(l *nvcdilib) {
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
	return func(l *nvcdilib) {
		l.nvidiaCDIHookPath = path
	}
}

// WithLdconfigPath sets the path to the ldconfig program
func WithLdconfigPath(path string) Option {
	return func(l *nvcdilib) {
		l.ldconfigPath = path
	}
}

// WithNvmlLib sets the nvml library for the library
func WithNvmlLib(nvmllib nvml.Interface) Option {
	return func(l *nvcdilib) {
		l.nvmllib = nvmllib
	}
}

// WithMode sets the discovery mode for the library
func WithMode[m modeConstraint](mode m) Option {
	return func(l *nvcdilib) {
		l.mode = Mode(mode)
	}
}

// WithVendor sets the vendor for the library
func WithVendor(vendor string) Option {
	return func(o *nvcdilib) {
		o.vendor = vendor
	}
}

// WithClass sets the class for the library
func WithClass(class string) Option {
	return func(o *nvcdilib) {
		o.class = class
	}
}

// WithMergedDeviceOptions sets the merged device options for the library
// If these are not set, no merged device will be generated.
func WithMergedDeviceOptions(opts ...transform.MergedDeviceOption) Option {
	return func(o *nvcdilib) {
		o.mergedDeviceOptions = opts
	}
}

// WithCSVFiles sets the CSV files for the library
func WithCSVFiles(csvFiles []string) Option {
	return func(o *nvcdilib) {
		o.csvFiles = csvFiles
	}
}

// WithCSVIgnorePatterns sets the ignore patterns for entries in the CSV files.
func WithCSVIgnorePatterns(csvIgnorePatterns []string) Option {
	return func(o *nvcdilib) {
		o.csvIgnorePatterns = csvIgnorePatterns
	}
}

// WithConfigSearchPaths sets the search paths for config files.
func WithConfigSearchPaths(paths []string) Option {
	return func(o *nvcdilib) {
		o.configSearchPaths = paths
	}
}

// WithLibrarySearchPaths sets the library search paths.
// This is currently only used for CSV-mode.
func WithLibrarySearchPaths(paths []string) Option {
	return func(o *nvcdilib) {
		o.librarySearchPaths = paths
	}
}

// WithDisabledHooks allows specific hooks to be disabled.
func WithDisabledHooks[T string | HookName](hooks ...T) Option {
	return func(o *nvcdilib) {
		for _, hook := range hooks {
			o.disabledHooks = append(o.disabledHooks, discover.HookName(hook))
		}
	}
}

// WithEnabledHooks explicitly enables a specific set of hooks.
// If a hook is explicitly enabled, this takes precedence over it being disabled.
func WithEnabledHooks[T string | HookName](hooks ...T) Option {
	return func(o *nvcdilib) {
		for _, hook := range hooks {
			o.enabledHooks = append(o.enabledHooks, discover.HookName(hook))
		}
	}
}

// WithFeatureFlags allows the specified set of features to be toggled on.
func WithFeatureFlags[T string | FeatureFlag](featureFlags ...T) Option {
	return func(o *nvcdilib) {
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
