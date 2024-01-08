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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// infoInterface provides an alias for mocking.
//
//go:generate moq -stub -out info-interface_mock.go . infoInterface
type infoInterface interface {
	info.Interface
	// UsesNVGPUModule indicates whether the system is using the nvgpu kernel module
	UsesNVGPUModule() (bool, string)
}

type resolver struct {
	logger logger.Interface
	info   infoInterface
}

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string, image image.CUDA) (rmode string) {
	nvinfo := info.New()
	nvmllib := nvml.New()
	devicelib := device.New(
		device.WithNvml(nvmllib),
	)

	info := additionalInfo{
		Interface: nvinfo,
		nvmllib:   nvmllib,
		devicelib: devicelib,
	}

	r := resolver{
		logger: logger,
		info:   info,
	}
	return r.resolveMode(mode, image)
}

// resolveMode determines the correct mode for the platform if set to "auto"
func (r resolver) resolveMode(mode string, image image.CUDA) (rmode string) {
	if mode != "auto" {
		r.logger.Infof("Using requested mode '%s'", mode)
		return mode
	}
	defer func() {
		r.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	if image.OnlyFullyQualifiedCDIDevices() {
		return "cdi"
	}

	isTegra, reason := r.info.IsTegraSystem()
	r.logger.Debugf("Is Tegra-based system? %v: %v", isTegra, reason)

	hasNVML, reason := r.info.HasNvml()
	r.logger.Debugf("Has NVML? %v: %v", hasNVML, reason)

	usesNVGPUModule, reason := r.info.UsesNVGPUModule()
	r.logger.Debugf("Uses nvgpu kernel module? %v: %v", usesNVGPUModule, reason)

	if (isTegra && !hasNVML) || usesNVGPUModule {
		return "csv"
	}

	return "legacy"
}
