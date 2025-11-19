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

// A RuntimeMode is used to select a specific mode of operation for the NVIDIA Container Runtime.
type RuntimeMode string

const (
	// In LegacyRuntimeMode the nvidia-container-runtime injects the
	// nvidia-container-runtime-hook as a prestart hook into the incoming
	// container config. This hook invokes the nvidia-container-cli to perform
	// the required modifications to the container.
	LegacyRuntimeMode = RuntimeMode("legacy")
	// In CSVRuntimeMode the nvidia-container-runtime processes a set of CSV
	// files to determine which container modification are required. The
	// contents of these CSV files are used to generate an in-memory CDI
	// specification which is used to modify the container config.
	CSVRuntimeMode = RuntimeMode("csv")
	// In CDIRuntimeMode the nvidia-container-runtime applies the modifications
	// to the container config required for the requested CDI devices in the
	// same way that other CDI clients would.
	CDIRuntimeMode = RuntimeMode("cdi")
	// In JitCDIRuntimeMode the nvidia-container-runtime generates in-memory CDI
	// specifications for requested NVIDIA devices.
	JitCDIRuntimeMode = RuntimeMode("jit-cdi")
)

type RuntimeModeResolver interface {
	ResolveRuntimeMode(string) RuntimeMode
}

type modeResolver struct {
	logger logger.Interface
	// TODO: This only needs to consider the requested devices.
	image                       *image.CUDA
	propertyExtractor           info.PropertyExtractor
	defaultMode                 RuntimeMode
	forceCSVModeForTegraSystems bool
}

type Option func(*modeResolver)

func WithDefaultMode(defaultMode RuntimeMode) Option {
	return func(mr *modeResolver) {
		mr.defaultMode = defaultMode
	}
}

func WithForceCSVModeForTegraSystems(forceCSVModeForTegraSystems bool) Option {
	return func(mr *modeResolver) {
		mr.forceCSVModeForTegraSystems = forceCSVModeForTegraSystems
	}
}

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
	r := &modeResolver{
		defaultMode: JitCDIRuntimeMode,
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.logger == nil {
		r.logger = &logger.NullLogger{}
	}

	return r
}

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string, image image.CUDA) (rmode RuntimeMode) {
	r := modeResolver{
		logger:            logger,
		image:             &image,
		propertyExtractor: nil,
	}
	return r.ResolveRuntimeMode(mode)
}

func (m *modeResolver) ResolveRuntimeMode(mode string) (rmode RuntimeMode) {
	if mode != "auto" {
		m.logger.Infof("Using requested mode '%s'", mode)
		return RuntimeMode(mode)
	}
	defer func() {
		m.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	if m.image.OnlyFullyQualifiedCDIDevices() {
		return CDIRuntimeMode
	}

	nvinfo := info.New(
		info.WithLogger(m.logger),
		info.WithPropertyExtractor(m.propertyExtractor),
	)

	switch nvinfo.ResolvePlatform() {
	case info.PlatformNVML, info.PlatformWSL:
		return m.defaultMode
	case info.PlatformTegra:
		if m.forceCSVModeForTegraSystems {
			return CSVRuntimeMode
		}
		return JitCDIRuntimeMode
	}
	return m.defaultMode
}
