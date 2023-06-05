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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/info"
)

// infoInterface provides an alias for mocking.
//
//go:generate moq -stub -out info-interface_mock.go . infoInterface
type infoInterface info.Interface

type resolver struct {
	logger logger.Interface
	info   info.Interface
}

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger logger.Interface, mode string) (rmode string) {
	nvinfo := info.New()
	r := resolver{
		logger: logger,
		info:   nvinfo,
	}
	return r.resolveMode(mode)
}

// resolveMode determines the correct mode for the platform if set to "auto"
func (r resolver) resolveMode(mode string) (rmode string) {
	if mode != "auto" {
		return mode
	}
	defer func() {
		r.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	isTegra, reason := r.info.IsTegraSystem()
	r.logger.Debugf("Is Tegra-based system? %v: %v", isTegra, reason)

	hasNVML, reason := r.info.HasNvml()
	r.logger.Debugf("Has NVML? %v: %v", hasNVML, reason)

	if isTegra && !hasNVML {
		return "csv"
	}

	return "legacy"
}
