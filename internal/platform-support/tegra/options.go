/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package tegra

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

type options struct {
	logger             logger.Interface
	driverRoot         string
	devRoot            string
	hookCreator        discover.HookCreator
	ldconfigPath       string
	librarySearchPaths []string

	// The following can be overridden for testing
	symlinkLocator      lookup.Locator
	symlinkChainLocator lookup.Locator
	// TODO: This should be replaced by a regular mock
	resolveSymlink func(string) (string, error)

	mountSpecs MountSpecPathsByTyper
}

// Option defines a functional option for configuring a Tegra discoverer.
type Option func(*options)

// WithLogger sets the logger for the discoverer.
func WithLogger(logger logger.Interface) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithDriverRoot sets the driver root for the discoverer.
func WithDriverRoot(driverRoot string) Option {
	return func(o *options) {
		o.driverRoot = driverRoot
	}
}

// WithDevRoot sets the /dev root.
// If this is unset, the driver root is assumed.
func WithDevRoot(devRoot string) Option {
	return func(o *options) {
		o.devRoot = devRoot
	}
}

// WithHookCreator sets the hook creator for the discoverer.
func WithHookCreator(hookCreator discover.HookCreator) Option {
	return func(o *options) {
		o.hookCreator = hookCreator
	}
}

// WithLdconfigPath sets the path to the ldconfig program
func WithLdconfigPath(ldconfigPath string) Option {
	return func(o *options) {
		o.ldconfigPath = ldconfigPath
	}
}

// WithLibrarySearchPaths sets the library search paths for the discoverer.
func WithLibrarySearchPaths(librarySearchPaths ...string) Option {
	return func(o *options) {
		o.librarySearchPaths = librarySearchPaths
	}
}

func WithMountSpecs(mountSpecs ...MountSpecPathsByTyper) Option {
	return func(o *options) {
		o.mountSpecs = mountSpecPathsByTypers(mountSpecs)
	}
}
