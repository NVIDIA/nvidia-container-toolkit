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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type Factory struct {
	logger      logger.Interface
	cfg         *config.Config
	driver      *root.Driver
	hookCreator discover.HookCreator
	image       *image.CUDA
	runtimeMode info.RuntimeMode
}

func NewFactory(opts ...Option) *Factory {
	f := &Factory{}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Create a modifier based on the modifier factory configuration.
func (f *Factory) Create() (oci.SpecModifier, error) {
	var modifiers List
	for _, modifierType := range supportedModifierTypes(f.runtimeMode) {
		switch modifierType {
		case "mode":
			modeModifier, err := f.newModeModifier()
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, modeModifier)
		case "nvidia-hook-remover":
			modifiers = append(modifiers, f.NewNvidiaContainerRuntimeHookRemover())
		case "graphics":
			graphicsModifier, err := f.NewGraphicsModifier()
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, graphicsModifier)
		case "feature-gated":
			featureGatedModifier, err := f.NewFeatureGatedModifier()
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, featureGatedModifier)
		default:
			f.logger.Debugf("Ignoring unknown modifier type %q", modifierType)
		}
	}
	return modifiers, nil
}

type Option func(*Factory)

func WithConfig(cfg *config.Config) Option {
	return func(f *Factory) {
		f.cfg = cfg
	}
}

func WithDriver(driver *root.Driver) Option {
	return func(f *Factory) {
		f.driver = driver
	}
}
func WithHookCreator(hookCreator discover.HookCreator) Option {
	return func(f *Factory) {
		f.hookCreator = hookCreator
	}
}

func WithImage(image *image.CUDA) Option {
	return func(f *Factory) {
		f.image = image
	}
}

func WithLogger(logger logger.Interface) Option {
	return func(f *Factory) {
		f.logger = logger
	}
}

func WithRuntimeMode(runtimeMode info.RuntimeMode) Option {
	return func(f *Factory) {
		f.runtimeMode = runtimeMode
	}
}
