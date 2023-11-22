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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// NewGraphicsModifier constructs a modifier that injects graphics-related modifications into an OCI runtime specification.
// The value of the NVIDIA_DRIVER_CAPABILITIES environment variable is checked to determine if this modification should be made.
func NewGraphicsModifier(logger logger.Interface, cfg *config.Config, image image.CUDA) (oci.SpecModifier, error) {
	if required, reason := requiresGraphicsModifier(image); !required {
		logger.Infof("No graphics modifier required: %v", reason)
		return nil, nil
	}

	// TODO: We should not just pass `nil` as the search path here.
	driver := root.New(logger, cfg.NVIDIAContainerCLIConfig.Root, nil)
	nvidiaCTKPath := cfg.NVIDIACTKConfig.Path

	mounts, err := discover.NewGraphicsMountsDiscoverer(
		logger,
		driver,
		nvidiaCTKPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mounts discoverer: %v", err)
	}

	// In standard usage, the devRoot is the same as the driver.Root.
	devRoot := driver.Root
	drmNodes, err := discover.NewDRMNodesDiscoverer(
		logger,
		image.DevicesFromEnvvars(visibleDevicesEnvvar),
		devRoot,
		nvidiaCTKPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct discoverer: %v", err)
	}

	d := discover.Merge(
		drmNodes,
		mounts,
	)
	return NewModifierFromDiscoverer(logger, d)
}

// requiresGraphicsModifier determines whether a graphics modifier is required.
func requiresGraphicsModifier(cudaImage image.CUDA) (bool, string) {
	if devices := cudaImage.DevicesFromEnvvars(visibleDevicesEnvvar); len(devices.List()) == 0 {
		return false, "no devices requested"
	}

	if !cudaImage.GetDriverCapabilities().Any(image.DriverCapabilityGraphics, image.DriverCapabilityDisplay) {
		return false, "no required capabilities requested"
	}

	return true, ""
}
