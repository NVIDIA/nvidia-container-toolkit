/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package info

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
)

// Option defines a function for passing options to the New() call.
type Option func(*options)

// WithDeviceLib sets the device library for the library.
func WithDeviceLib(devicelib device.Interface) Option {
	return func(i *options) {
		i.devicelib = devicelib
	}
}

// WithLogger sets the logger for the library.
func WithLogger(logger basicLogger) Option {
	return func(i *options) {
		i.logger = logger
	}
}

// WithNvmlLib sets the nvml library for the library.
func WithNvmlLib(nvmllib nvml.Interface) Option {
	return func(i *options) {
		i.nvmllib = nvmllib
	}
}

// WithRoot provides a Option to set the root of the 'info' interface.
func WithRoot(r string) Option {
	return func(i *options) {
		i.root = root(r)
	}
}

// WithPropertyExtractor provides an Option to set the PropertyExtractor
// interface implementation.
// This is predominantly used for testing.
func WithPropertyExtractor(propertyExtractor PropertyExtractor) Option {
	return func(i *options) {
		i.propertyExtractor = propertyExtractor
	}
}

// WithPlatform provides an option to set the platform explicitly.
func WithPlatform(platform Platform) Option {
	return func(i *options) {
		i.platform = platform
	}
}
