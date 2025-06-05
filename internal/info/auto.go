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
	RuntimeModeLegacy = RuntimeMode("legacy")
	RuntimeModeCSV    = RuntimeMode("csv")
	RuntimeModeCDI    = RuntimeMode("cdi")
	RuntimeModeJitCDI = RuntimeMode("jit-cdi")
)

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string, image image.CUDA) (rmode RuntimeMode) {
	return resolveMode(logger, mode, image, nil)
}

func resolveMode(logger logger.Interface, mode string, image image.CUDA, propertyExtractor info.PropertyExtractor) (rmode RuntimeMode) {
	if mode != "auto" {
		logger.Infof("Using requested mode '%s'", mode)
		return RuntimeMode(mode)
	}
	defer func() {
		logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	if image.OnlyFullyQualifiedCDIDevices() {
		return RuntimeModeCDI
	}

	nvinfo := info.New(
		info.WithLogger(logger),
		info.WithPropertyExtractor(propertyExtractor),
	)

	switch nvinfo.ResolvePlatform() {
	case info.PlatformNVML, info.PlatformWSL:
		return RuntimeModeJitCDI
	case info.PlatformTegra:
		return RuntimeModeCSV
	}
	return RuntimeModeJitCDI
}
