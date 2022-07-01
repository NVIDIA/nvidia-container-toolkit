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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/sirupsen/logrus"
)

const (
	nvidiaMOFEDEnvvar = "NVIDIA_MOFED"
)

// NewMOFEDModifier creates the modifiers for MOFED devices.
// If the spec does not contain the NVIDIA_MOFED=enabled environment variable no changes are made.
func NewMOFEDModifier(logger *logrus.Logger, cfg *config.Config, ociSpec oci.Spec) (oci.SpecModifier, error) {
	_, err := ociSpec.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	// We check whether a modification is required and return a nil modifier if this is not the case.
	visibleDevices, exists := ociSpec.LookupEnv(visibleDevicesEnvvar)
	if !exists || visibleDevices == "" || visibleDevices == visibleDevicesVoid {
		logger.Infof("No modification required: %v=%v (exists=%v)", visibleDevicesEnvvar, visibleDevices, exists)
		return nil, nil
	}

	if mofed, _ := ociSpec.LookupEnv(nvidiaMOFEDEnvvar); mofed != "enabled" {
		return nil, nil
	}

	d, err := discover.NewMOFEDDiscoverer(logger, cfg.NVIDIAContainerCLIConfig.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to construct discoverer for MOFED devices: %v", err)
	}

	return NewModifierFromDiscoverer(logger, d)
}
