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
	"maps"
	"slices"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// fromRegistry represents the modifications performed using a CDI registry.
type fromRegistry struct {
	logger   logger.Interface
	registry *cdi.Cache
	devices  []string

	// deviceRequestSource is only used to explain resolution failures.
	// See unresolvedDeviceHint.
	deviceRequestSource string
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

		if hint := m.unresolvedDeviceHint(unresolvedDevices); hint != "" {
			return fmt.Errorf("failed to inject CDI devices: %v: %s", err, hint)
		}
		return fmt.Errorf("failed to inject CDI devices: %v", err)
	}

	return nil
}

// unresolvedDeviceHint describes why the specified device names could not be
// resolved, for the causes that the registry contents can distinguish.
//
// A device name requested without a kind is qualified with default-kind before
// it reaches the registry, so a name that resolves under a different kind means
// the device list and default-kind disagree about which spec the devices came
// from. Only the mechanism that supplied the list is reported: default-kind is
// also what management containers requesting "all" rely on, so it is rarely the
// side that should change.
func (m fromRegistry) unresolvedDeviceHint(unresolved []string) string {
	kindsByDevice := make(map[string][]string)
	kinds := make(map[string]bool)
	for _, available := range m.registry.ListDevices() {
		vendor, class, name, err := parser.ParseQualifiedName(available)
		if err != nil {
			continue
		}
		kind := vendor + "/" + class
		kinds[kind] = true
		kindsByDevice[name] = append(kindsByDevice[name], kind)
	}

	for _, device := range unresolved {
		requestedKind, name, isQualified := strings.Cut(device, "=")
		if !isQualified {
			continue
		}
		alternatives := slices.DeleteFunc(slices.Clone(kindsByDevice[name]), func(kind string) bool {
			return kind == requestedKind
		})
		if len(alternatives) > 0 {
			return "device available as kind " + strings.Join(alternatives, ", ") + m.requestedVia()
		}
	}

	if len(kinds) > 0 {
		return "registry contains only kinds " + strings.Join(slices.Sorted(maps.Keys(kinds)), ", ")
	}
	return fmt.Sprintf("no CDI specifications found in %v", m.registry.GetSpecDirectories())
}

// requestedVia names the mechanism that supplied the device list, which is the
// component to correct. Every mechanism can express a fully-qualified name, so
// the mechanism itself is never the fault; it is reported to locate the caller
// that sent a name the registry could not resolve.
func (m fromRegistry) requestedVia() string {
	if m.deviceRequestSource == "" {
		return ""
	}
	return "; requested via " + m.deviceRequestSource
}
