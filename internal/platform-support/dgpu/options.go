/**
# Copyright 2024 NVIDIA CORPORATION
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

package dgpu

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
)

type options struct {
	logger            logger.Interface
	devRoot           string
	nvidiaCDIHookPath string

	isMigDevice bool
	// migCaps stores the MIG capabilities for the system.
	// If MIG is not available, this is nil.
	migCaps      nvcaps.MigCaps
	migCapsError error

	nvsandboxutilslib nvsandboxutils.Interface
}

type Option func(*options)

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

// WithNVIDIACDIHookPath sets the path to the NVIDIA Container Toolkit CLI path for the library
func WithNVIDIACDIHookPath(path string) Option {
	return func(l *options) {
		l.nvidiaCDIHookPath = path
	}
}

// WithMIGCaps sets the MIG capabilities.
func WithMIGCaps(migCaps nvcaps.MigCaps) Option {
	return func(l *options) {
		l.migCaps = migCaps
	}
}

// WithNvsandboxuitilsLib sets the nvsandboxutils library implementation.
func WithNvsandboxuitilsLib(nvsandboxutilslib nvsandboxutils.Interface) Option {
	return func(l *options) {
		l.nvsandboxutilslib = nvsandboxutilslib
	}
}
