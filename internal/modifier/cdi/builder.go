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

package cdi

import (
	"fmt"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type builder struct {
	logger   logger.Interface
	specDirs []string
	devices  []string
	cdiSpec  *specs.Spec
}

// Option represents a functional option for creating a CDI mofifier.
type Option func(*builder)

// New creates a new CDI modifier.
func New(opts ...Option) (oci.SpecModifier, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logger.New()
	}
	return b.build()
}

// build uses the applied options and constructs a CDI modifier using the builder.
func (m builder) build() (oci.SpecModifier, error) {
	if len(m.devices) == 0 && m.cdiSpec == nil {
		return nil, nil
	}

	if m.cdiSpec != nil {
		modifier := fromCDISpec{
			cdiSpec: &cdi.Spec{Spec: m.cdiSpec},
		}
		return modifier, nil
	}

	registry, err := cdi.NewCache(
		cdi.WithAutoRefresh(false),
		cdi.WithSpecDirs(m.specDirs...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDI registry: %v", err)
	}

	modifier := fromRegistry{
		logger:   m.logger,
		registry: registry,
		devices:  m.devices,
	}

	return modifier, nil
}

// WithLogger sets the logger for the CDI modifier builder.
func WithLogger(logger logger.Interface) Option {
	return func(b *builder) {
		b.logger = logger
	}
}

// WithSpecDirs sets the spec directories for the CDI modifier builder.
func WithSpecDirs(specDirs ...string) Option {
	return func(b *builder) {
		b.specDirs = specDirs
	}
}

// WithDevices sets the devices for the CDI modifier builder.
func WithDevices(devices ...string) Option {
	return func(b *builder) {
		b.devices = devices
	}
}

// WithSpec sets the spec for the CDI modifier builder.
func WithSpec(spec *specs.Spec) Option {
	return func(b *builder) {
		b.cdiSpec = spec
	}
}
