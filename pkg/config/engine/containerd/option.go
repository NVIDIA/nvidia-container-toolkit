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

package containerd

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

type builder struct {
	logger               logger.Interface
	configSource         toml.Loader
	configVersion        int
	useLegacyConfig      bool
	path                 string
	runtimeType          string
	containerAnnotations []string
}

// Option defines a function that can be used to configure the config builder
type Option func(*builder)

// WithLogger sets the logger for the config builder
func WithLogger(logger logger.Interface) Option {
	return func(b *builder) {
		b.logger = logger
	}
}

// WithPath sets the path for the config builder
func WithPath(path string) Option {
	return func(b *builder) {
		b.path = path
	}
}

// WithConfigSource sets the source for the config.
func WithConfigSource(configSource toml.Loader) Option {
	return func(b *builder) {
		b.configSource = configSource
	}
}

// WithRuntimeType sets the runtime type for the config builder
func WithRuntimeType(runtimeType string) Option {
	return func(b *builder) {
		b.runtimeType = runtimeType
	}
}

// WithUseLegacyConfig sets the useLegacyConfig flag for the config builder.
func WithUseLegacyConfig(useLegacyConfig bool) Option {
	return func(b *builder) {
		b.useLegacyConfig = useLegacyConfig
	}
}

// WithConfigVersion sets the config version for the config builder
func WithConfigVersion(configVersion int) Option {
	return func(b *builder) {
		b.configVersion = configVersion
	}
}

// WithContainerAnnotations sets the container annotations for the config builder
func WithContainerAnnotations(containerAnnotations ...string) Option {
	return func(b *builder) {
		b.containerAnnotations = containerAnnotations
	}
}
