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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
)

// Option defines a function for passing options to the New() call
type Option func(*builder)

// WithDeviceLib sets the device library for the library
func WithDeviceLib(devicelib device.Interface) Option {
	return func(l *builder) {
		l.devicelib = devicelib
	}
}

// WithLogger sets the logger for the library.
func WithLogger(logger basicLogger) Option {
	return func(i *builder) {
		i.logger = logger
	}
}

// WithNvmlLib sets the nvml library for the library
func WithNvmlLib(nvmllib nvml.Interface) Option {
	return func(l *builder) {
		l.nvmllib = nvmllib
	}
}

// WithRoot provides a Option to set the root of the 'info' interface
func WithRoot(root string) Option {
	return func(i *builder) {
		i.root = root
	}
}

// WithPreHookResolver provides an Option to set resolvers to use before others.
func WithPreHookResolver(preHook Resolver) Option {
	return func(i *builder) {
		i.preHook = preHook
	}
}

// WithProperties provides an Option to set the Properties interface implementation.
// This is predominantly used for testing.
func WithProperties(properties Properties) Option {
	return func(i *builder) {
		i.properties = properties
	}
}
