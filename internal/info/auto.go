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

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string, image image.CUDA) (rmode string) {
	return resolveMode(logger, mode, image, nil)
}

func resolveMode(logger logger.Interface, mode string, image image.CUDA, propertyExtractor info.PropertyExtractor) (rmode string) {
	if mode != "auto" {
		logger.Infof("Using requested mode '%s'", mode)
		return mode
	}
	defer func() {
		logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	if image.OnlyFullyQualifiedCDIDevices() {
		return "cdi"
	}

	nvinfo := info.New(
		info.WithLogger(logger),
		info.WithPropertyExtractor(propertyExtractor),
	)

	switch nvinfo.ResolvePlatform() {
	case info.PlatformNVML, info.PlatformWSL:
		return "legacy"
	case info.PlatformTegra:
		return "csv"
	}
	return "legacy"
}
