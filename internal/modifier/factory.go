/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package modifier

import (
	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

type Factory struct {
	logger      logger.Interface
	cfg         *config.Config
	driver      *root.Driver
	hookCreator discover.HookCreator
	image       *image.CUDA

	editsFactory edits.Factory
}

type FactoryOption func(*Factory)

func NewFactory(opts ...FactoryOption) *Factory {
	f := &Factory{}
	for _, opt := range opts {
		opt(f)
	}

	if f.editsFactory == nil {
		f.editsFactory = edits.NewFactory(edits.WithLogger(f.logger))
	}

	return f
}

func WithLogger(logger logger.Interface) FactoryOption {
	return func(f *Factory) {
		f.logger = logger
	}
}

func WithConfig(cfg *config.Config) FactoryOption {
	return func(f *Factory) {
		f.cfg = cfg
	}
}

func WithImage(image *image.CUDA) FactoryOption {
	return func(f *Factory) {
		f.image = image
	}
}

func WithDriver(driver *root.Driver) FactoryOption {
	return func(f *Factory) {
		f.driver = driver
	}
}

func WithHookCreator(hookCreator discover.HookCreator) FactoryOption {
	return func(f *Factory) {
		f.hookCreator = hookCreator
	}
}
