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

package modifier

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	nvidiaGDSEnvvar      = "NVIDIA_GDS"
	nvidiaMOFEDEnvvar    = "NVIDIA_MOFED"
	nvidiaNVSWITCHEnvvar = "NVIDIA_NVSWITCH"
)

// NewFeatureGatedModifier creates the modifiers for optional features.
// These include:
//
//	NVIDIA_GDS=enabled
//	NVIDIA_MOFED=enabled
//	NVIDIA_NVSWITCH=enabled
//
// If not devices are selected, no changes are made.
func NewFeatureGatedModifier(logger logger.Interface, cfg *config.Config, image image.CUDA) (oci.SpecModifier, error) {
	if devices := image.DevicesFromEnvvars(visibleDevicesEnvvar); len(devices.List()) == 0 {
		logger.Infof("No modification required; no devices requested")
		return nil, nil
	}

	var discoverers []discover.Discover

	driverRoot := cfg.NVIDIAContainerCLIConfig.Root
	devRoot := cfg.NVIDIAContainerCLIConfig.Root

	if image.Getenv(nvidiaGDSEnvvar) == "enabled" {
		d, err := discover.NewGDSDiscoverer(logger, driverRoot, devRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to construct discoverer for GDS devices: %w", err)
		}
		discoverers = append(discoverers, d)
	}

	if image.Getenv(nvidiaMOFEDEnvvar) == "enabled" {
		d, err := discover.NewMOFEDDiscoverer(logger, devRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to construct discoverer for MOFED devices: %w", err)
		}
		discoverers = append(discoverers, d)
	}

	if image.Getenv(nvidiaNVSWITCHEnvvar) == "enabled" {
		d, err := discover.NewNvSwitchDiscoverer(logger, devRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to construct discoverer for NVSWITCH devices: %w", err)
		}
		discoverers = append(discoverers, d)
	}

	return NewModifierFromDiscoverer(logger, discover.Merge(discoverers...))
}
