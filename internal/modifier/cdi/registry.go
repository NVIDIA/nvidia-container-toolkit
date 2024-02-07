/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package cdi

import (
	"errors"
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// fromRegistry represents the modifications performed using a CDI registry.
type fromRegistry struct {
	logger   logger.Interface
	registry *cdi.Cache
	devices  []string
}

var _ oci.SpecModifier = (*fromRegistry)(nil)

// Modify applies the modifications defined by the CDI registry to the incoming OCI spec.
func (m fromRegistry) Modify(spec *specs.Spec) error {
	if err := m.registry.Refresh(); err != nil {
		m.logger.Debugf("The following error was triggered when refreshing the CDI registry: %v", err)
	}

	m.logger.Debugf("Injecting devices using CDI: %v", m.devices)
	unresolvedDevices, err := m.registry.InjectDevices(spec, m.devices...)
	if unresolvedDevices != nil {
		m.logger.Warningf("could not resolve CDI devices: %v", unresolvedDevices)
	}
	if err != nil {
		var refreshErrors []error
		for _, rerrs := range m.registry.GetErrors() {
			refreshErrors = append(refreshErrors, rerrs...)
		}
		if rerr := errors.Join(refreshErrors...); rerr != nil {
			// We log the errors that may have been generated while refreshing the CDI registry.
			// These may be due to malformed specifications or device name conflicts that could be
			// the cause of an injection failure.
			m.logger.Warningf("Refreshing the CDI registry generated errors: %v", rerr)
		}

		return fmt.Errorf("failed to inject CDI devices: %v", err)
	}

	return nil
}
