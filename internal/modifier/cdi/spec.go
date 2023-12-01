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
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// fromCDISpec represents the modifications performed from a raw CDI spec.
type fromCDISpec struct {
	cdiSpec *cdi.Spec
}

var _ oci.SpecModifier = (*fromCDISpec)(nil)

// Modify applies the mofiications defined by the raw CDI spec to the incomming OCI spec.
func (m fromCDISpec) Modify(spec *specs.Spec) error {
	for _, device := range m.cdiSpec.Devices {
		device := device
		cdiDevice := cdi.Device{
			Device: &device,
		}
		if err := cdiDevice.ApplyEdits(spec); err != nil {
			return fmt.Errorf("failed to apply edits for device %q: %v", cdiDevice.GetQualifiedName(), err)
		}
	}

	return m.cdiSpec.ApplyEdits(spec)
}
