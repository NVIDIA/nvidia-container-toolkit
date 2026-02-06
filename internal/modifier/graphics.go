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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// NewGraphicsModifier constructs a modifier that injects graphics-related modifications into an OCI runtime specification.
// The value of the NVIDIA_DRIVER_CAPABILITIES environment variable is checked to determine if this modification should be made.
func (f *Factory) NewGraphicsModifier() (oci.SpecModifier, error) {
	devices, reason := requiresGraphicsModifier(*f.image)
	if len(devices) == 0 {
		f.logger.Infof("No graphics modifier required; %v", reason)
		return nil, nil
	}

	mounts, err := discover.NewGraphicsMountsDiscoverer(
		f.logger,
		f.driver,
		f.hookCreator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mounts discoverer: %v", err)
	}

	// In standard usage, the devRoot is the same as the driver.Root.
	devRoot := f.driver.Root
	drmNodes, err := discover.NewDRMNodesDiscoverer(
		f.logger,
		image.NewVisibleDevices(devices...),
		devRoot,
		f.hookCreator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct discoverer: %v", err)
	}

	d := discover.Merge(
		drmNodes,
		mounts,
	)
	return f.NewModifierFromDiscoverer(d)
}

// requiresGraphicsModifier determines whether a graphics modifier is required.
func requiresGraphicsModifier(cudaImage image.CUDA) ([]string, string) {
	devices := cudaImage.VisibleDevices()
	if len(devices) == 0 {
		return nil, "no devices requested"
	}

	if !cudaImage.GetDriverCapabilities().Any(image.DriverCapabilityGraphics, image.DriverCapabilityDisplay) {
		return nil, "no required capabilities requested"
	}

	return devices, ""
}
