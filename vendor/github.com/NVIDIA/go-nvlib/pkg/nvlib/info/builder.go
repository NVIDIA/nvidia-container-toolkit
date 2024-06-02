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
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
)

type infolib struct {
	PropertyExtractor
	PlatformResolver
}

type options struct {
	logger    basicLogger
	root      root
	nvmllib   nvml.Interface
	devicelib device.Interface

	platform          Platform
	propertyExtractor PropertyExtractor
}

// New creates a new instance of the 'info' interface.
func New(opts ...Option) Interface {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = &nullLogger{}
	}
	if o.root == "" {
		o.root = "/"
	}
	if o.nvmllib == nil {
		o.nvmllib = nvml.New(
			nvml.WithLibraryPath(o.root.tryResolveLibrary("libnvidia-ml.so.1")),
		)
	}
	if o.devicelib == nil {
		o.devicelib = device.New(o.nvmllib)
	}
	if o.platform == "" {
		o.platform = PlatformAuto
	}
	if o.propertyExtractor == nil {
		o.propertyExtractor = &propertyExtractor{
			root:      o.root,
			nvmllib:   o.nvmllib,
			devicelib: o.devicelib,
		}
	}
	return &infolib{
		PlatformResolver: &platformResolver{
			logger:            o.logger,
			platform:          o.platform,
			propertyExtractor: o.propertyExtractor,
		},
		PropertyExtractor: o.propertyExtractor,
	}
}
