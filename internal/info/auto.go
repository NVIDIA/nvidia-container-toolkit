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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type RuntimeModeResolver interface {
	ResolveRuntimeMode(string) string
}

type modeResolver struct {
	logger logger.Interface
	// TODO: This only needs to consider the requested devices.
	image             *image.CUDA
	propertyExtractor info.PropertyExtractor
}

type Option func(*modeResolver)

func WithLogger(logger logger.Interface) Option {
	return func(mr *modeResolver) {
		mr.logger = logger
	}
}

func WithImage(image *image.CUDA) Option {
	return func(mr *modeResolver) {
		mr.image = image
	}
}

func WithPropertyExtractor(propertyExtractor info.PropertyExtractor) Option {
	return func(mr *modeResolver) {
		mr.propertyExtractor = propertyExtractor
	}
}

func NewRuntimeModeResolver(opts ...Option) RuntimeModeResolver {
	r := &modeResolver{}
	for _, opt := range opts {
		opt(r)
	}
	if r.logger == nil {
		r.logger = &logger.NullLogger{}
	}

	return r
}

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string, image image.CUDA) (rmode string) {
	r := modeResolver{
		logger:            logger,
		image:             &image,
		propertyExtractor: nil,
	}
	return r.ResolveRuntimeMode(mode)
}

func (m *modeResolver) ResolveRuntimeMode(mode string) (rmode string) {
	if mode != "auto" {
		m.logger.Infof("Using requested mode '%s'", mode)
		return mode
	}
	defer func() {
		m.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	if m.image.OnlyFullyQualifiedCDIDevices() {
		return "cdi"
	}

	nvinfo := info.New(
		info.WithLogger(m.logger),
		info.WithPropertyExtractor(m.propertyExtractor),
	)

	switch nvinfo.ResolvePlatform() {
	case info.PlatformNVML, info.PlatformWSL:
		return "legacy"
	case info.PlatformTegra:
		return "csv"
	}
	return "legacy"
}
