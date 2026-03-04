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

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// newFeatureGatedModifier creates the modifiers for optional features.
// These include:
//
//	NVIDIA_GDS=enabled
//	NVIDIA_MOFED=enabled
//	NVIDIA_NVSWITCH=enabled
//	NVIDIA_GDRCOPY=enabled
//
// If not devices are selected, no changes are made.
func (f *Factory) newFeatureGatedModifier() (oci.SpecModifier, error) {
	if devices := f.image.VisibleDevices(); len(devices) == 0 {
		f.logger.Infof("No modification required; no devices requested")
		return nil, nil
	}

	var modifers list
	if gatedDeviceRequests := withUniqueDevices(gatedDevices(*f.image)).DeviceRequests(); len(gatedDeviceRequests) != 0 {
		featureGatedModifier, err := f.newAutomaticCDISpecModifier(gatedDeviceRequests)
		if err != nil {
			return nil, err
		}
		modifers = append(modifers, featureGatedModifier)
	}

	// If the feature flag has explicitly been toggled, we don't make any modification.
	if !f.cfg.Features.DisableCUDACompatLibHook.IsEnabled() {
		cudaCompatModifer, err := f.getCudaCompatModeModifier()
		if err != nil {
			return nil, fmt.Errorf("failed to construct CUDA Compat discoverer: %w", err)
		}
		modifers = append(modifers, cudaCompatModifer)
	}

	return modifers, nil
}

func (f *Factory) getCudaCompatModeModifier() (oci.SpecModifier, error) {
	// We don't support the enable-cuda-compat hook in CSV mode.
	if f.cfg.NVIDIAContainerRuntimeConfig.Mode == "csv" {
		return nil, nil
	}

	// For legacy mode, we only include the enable-cuda-compat hook if cuda-compat-mode is set to hook.
	if f.cfg.NVIDIAContainerRuntimeConfig.Mode == "legacy" && f.cfg.NVIDIAContainerRuntimeConfig.Modes.Legacy.CUDACompatMode != config.CUDACompatModeHook {
		return nil, nil
	}

	version, err := f.driver.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to get driver version: %w", err)
	}

	compatLibHookDiscoverer := discover.NewCUDACompatHookDiscoverer(f.logger, f.hookCreator, &discover.EnableCUDACompatHookOptions{HostDriverVersion: version})
	// For non-legacy modes we return the hook as is. These modes *should* already include the update-ldcache hook.
	if f.cfg.NVIDIAContainerRuntimeConfig.Mode != "legacy" {
		return f.newModifierFromDiscoverer(compatLibHookDiscoverer)
	}

	// For legacy mode, we also need to inject a hook to update the LDCache
	// after we have modifed the configuration.
	ldcacheUpdateHookDiscoverer, err := discover.NewLDCacheUpdateHook(
		f.logger,
		discover.None{},
		f.hookCreator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct ldcache update discoverer: %w", err)
	}

	return f.newModifierFromDiscoverer(discover.Merge(compatLibHookDiscoverer, ldcacheUpdateHookDiscoverer))
}
