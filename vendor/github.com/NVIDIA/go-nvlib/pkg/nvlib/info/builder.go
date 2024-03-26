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

package info

import (
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
)

type builder struct {
	logger    basicLogger
	root      string
	nvmllib   nvml.Interface
	devicelib device.Interface

	preHook    Resolver
	properties Properties
}

// New creates a new instance of the 'info' interface
func New(opts ...Option) Interface {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = &nullLogger{}
	}
	if b.root == "" {
		b.root = "/"
	}
	if b.nvmllib == nil {
		b.nvmllib = nvml.New()
	}
	if b.devicelib == nil {
		b.devicelib = device.New(device.WithNvml(b.nvmllib))
	}
	if b.preHook == nil {
		b.preHook = noop{}
	}
	if b.properties == nil {
		b.properties = &info{
			root:      b.root,
			nvmllib:   b.nvmllib,
			devicelib: b.devicelib,
		}
	}
	return b.build()
}

func (b *builder) build() Interface {
	return &infolib{
		logger:     b.logger,
		Resolver:   b.getResolvers(),
		Properties: b.properties,
	}
}

func (b *builder) getResolvers() Resolver {
	auto := &notEqualsResolver{
		logger: b.logger,
		mode:   "auto",
	}

	systemMode := &systemMode{
		logger:     b.logger,
		Properties: b.properties,
	}

	return firstOf([]Resolver{
		auto,
		b.preHook,
		systemMode,
	})
}
