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
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	cdi "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

type cdiModifier struct {
	logger  *logrus.Logger
	devices []string
}

// NewCDIModifier creates an OCI spec modifier that determines the modifications to make based on the
// CDI specifications available on the system. The NVIDIA_VISIBLE_DEVICES enviroment variable is
// used to select the devices to include.
func NewCDIModifier(logger *logrus.Logger, cfg *config.Config, ociSpec oci.Spec) (oci.SpecModifier, error) {
	rawSpec, err := ociSpec.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	image, err := image.NewCUDAImageFromSpec(rawSpec)
	if err != nil {
		return nil, err
	}

	devices := image.DevicesFromEnvvars(visibleDevicesEnvvar)
	if len(devices) == 0 {
		logger.Debugf("No modification required; no devices requested")
		return nil, nil
	}

	var qualifiedDevices []string
	for _, name := range devices {
		if !cdi.IsQualifiedName(name) {
			name = cdi.QualifiedName("nvidia.com", "gpu", name)
		}
		qualifiedDevices = append(qualifiedDevices, name)
	}

	m := cdiModifier{
		logger:  logger,
		devices: qualifiedDevices,
	}

	return m, nil
}

// Modify loads the CDI registry and injects the specified CDI devices into the OCI runtime specification.
func (m cdiModifier) Modify(spec *specs.Spec) error {
	registry := cdi.GetRegistry(
		cdi.WithAutoRefresh(false),
	)
	if errs := registry.GetErrors(); len(errs) > 0 {
		m.logger.Debugf("The following errors were triggered when creating the CDI registry: %v", errs)
	}

	devices := m.devices
	for _, d := range devices {
		if d == "nvidia.com/gpu=all" {
			devices = []string{}
			for _, candidate := range registry.DeviceDB().ListDevices() {
				if strings.HasPrefix(candidate, "nvidia.com/gpu=") {
					devices = append(devices, candidate)
				}
			}
			break
		}
	}

	_, err := registry.InjectDevices(spec, devices...)
	if err != nil {
		return fmt.Errorf("failed to inject CDI devices: %v", err)
	}

	return nil
}
