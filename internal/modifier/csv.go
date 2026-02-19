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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/cuda"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/requirements"
)

// newCSVModifier creates a modifier that applies modications to an OCI spec if required by the runtime wrapper.
// The modifications are defined by CSV MountSpecs.
func (f *Factory) newCSVModifier() (oci.SpecModifier, error) {
	devices := withUniqueDevices(csvDevices(*f.image)).DeviceRequests()
	if len(devices) == 0 {
		f.logger.Infof("No modification required; no devices requested")
		return nil, nil
	}
	f.logger.Infof("Constructing modifier from config: %+v", *f.cfg)

	if err := checkRequirements(f.logger, f.image); err != nil {
		return nil, fmt.Errorf("requirements not met: %v", err)
	}

	return f.newAutomaticCDISpecModifier(devices)
}

func checkRequirements(logger logger.Interface, image *image.CUDA) error {
	if image == nil || image.HasDisableRequire() {
		// TODO: We could print the real value here instead
		logger.Debugf("NVIDIA_DISABLE_REQUIRE=%v; skipping requirement checks", true)
		return nil
	}

	imageRequirements, err := image.GetRequirements()
	if err != nil {
		//  TODO: Should we treat this as a failure, or just issue a warning?
		return fmt.Errorf("failed to get image requirements: %v", err)
	}

	r := requirements.New(logger, imageRequirements)

	cudaVersion, err := cuda.Version()
	if err != nil {
		logger.Warningf("Failed to get CUDA version: %v", err)
	} else {
		r.AddVersionProperty(requirements.CUDA, cudaVersion)
	}

	compteCapability, err := cuda.ComputeCapability(0)
	if err != nil {
		logger.Warningf("Failed to get CUDA Compute Capability: %v", err)
	} else {
		r.AddVersionProperty(requirements.ARCH, compteCapability)
	}

	return r.Assert()
}

type csvDevices image.CUDA

func (d csvDevices) DeviceRequests() []string {
	var devices []string
	i := (image.CUDA)(d)
	for _, deviceID := range i.VisibleDevices() {
		devices = append(devices, "mode=csv,id="+deviceID)
	}
	return devices
}
