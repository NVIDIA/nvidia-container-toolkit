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
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdi "tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// fromCDISpec represents the modifications performed from a raw CDI spec.
type fromCDISpec struct {
	cdiSpec *cdiapi.Spec
}

var _ oci.SpecModifier = (*fromCDISpec)(nil)

// Modify applies the mofiications defined by the raw CDI spec to the incomming OCI spec.
func (m fromCDISpec) Modify(spec *specs.Spec) error {
	for _, device := range m.cdiSpec.Devices {
		device := m.enrichDevice(device)
		cdiDevice := cdiapi.Device{
			Device: &device,
		}
		if err := cdiDevice.ApplyEdits(spec); err != nil {
			return fmt.Errorf("failed to apply edits for device %q: %v", m.cdiSpec.Kind+"="+device.Name, err)
		}
	}

	return m.cdiSpec.ApplyEdits(spec)
}

func (m fromCDISpec) enrichDevice(device cdi.Device) cdi.Device {
	if !devices.IsOverrideApplied() {
		return device
	}
	// For testing we need to override the device node information to ensure
	// that we don't trigger the CDI modification that requires the device node
	// to exist and be a character device.
	// The following condition is used to determine whether a failure to get
	// the info is fatal:
	// hasMinimalSpecification := d.Type != "" && (d.Major != 0 || d.Type == fifoDevice)
	for i, dn := range device.ContainerEdits.DeviceNodes {
		dn.Type = "c"
		if dn.Major == 0 {
			dn.Major = 99
		}
		device.ContainerEdits.DeviceNodes[i] = dn
	}
	return device
}
