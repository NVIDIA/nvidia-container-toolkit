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
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
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

	editsFactory edits.Factory
}

// A Factory also implements the oci.SpecModifier interface.
var _ oci.SpecModifier = (*Factory)(nil)

// New is a factory method for creating a modifier Factory.
func New(opts ...Option) (oci.SpecModifier, error) {
	f := createFactory(opts...)
	if err := f.validate(); err != nil {
		return nil, err
	}
	return f, nil
}

// createFactory is an internal constructor to allow validation to be bypassed
// for internal tests.
func createFactory(opts ...Option) *Factory {
	f := &Factory{}
	for _, opt := range opts {
		opt(f)
	}

	if f.editsFactory == nil {
		f.editsFactory = edits.NewFactory(edits.WithLogger(f.logger))
	}

	return f
}

func (f *Factory) validate() error {
	switch string(f.runtimeMode) {
	case "":
		return fmt.Errorf("a mode must be specified")
	case "legacy", "csv", "jit-cdi", "cdi":
		return nil
	default:
		return fmt.Errorf("invalid mode %q", f.runtimeMode)
	}
}

// Modify creates the configured modifier and applies it to the supplied OCI
// specification.
func (f *Factory) Modify(s *specs.Spec) error {
	m, err := f.create()
	if err != nil {
		return err
	}
	return m.Modify(s)
}

// create a modifier based on the modifier factory configuration.
func (f *Factory) create() (oci.SpecModifier, error) {
	var modifiers list
	for _, modifierType := range supportedModifierTypes(f.runtimeMode) {
		switch modifierType {
		case "mode":
			modeModifier, err := f.newModeModifier()
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, modeModifier)
		case "nvidia-hook-remover":
			modifiers = append(modifiers, f.newNvidiaContainerRuntimeHookRemover())
		case "graphics":
			graphicsModifier, err := f.newGraphicsModifier()
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, graphicsModifier)
		case "feature-gated":
			featureGatedModifier, err := f.newFeatureGatedModifier()
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
