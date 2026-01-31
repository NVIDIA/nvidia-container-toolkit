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
	cdiSpec   *cdi.Spec
	mknodOnly bool
}

var _ oci.SpecModifier = (*fromCDISpec)(nil)

// Modify applies the mofications defined by the raw CDI spec to the incomming OCI spec.
func (m fromCDISpec) Modify(spec *specs.Spec) error {
	for _, device := range m.cdiSpec.Devices {
		device := device
		if m.mknodOnly {
			for i := range device.ContainerEdits.DeviceNodes {
				// We cannot set an empty string as this will be translated to rwm in the OCI spec generation
				// see here:
				// https://github.com/NVIDIA/nvidia-container-toolkit/blob/786aa3baf25bf0acd26ae48b5934fa5d503fa1ec/vendor/tags.cncf.io/container-device-interface/pkg/cdi/container-edits.go#L110
				// Since we use the CDI implem above and we can actually pass any arbitrary string, for some oci compatible
				// runtimes like runc we could set "_", and this would have the intended effect (no permissions) but
				// this relies on two successive flaws in the CDI/OCI specs implems
				// (non repect of the CDI spec in the implem above + non respect of the OCI spec in the underlying runtime)
				// Let's just add the mknod permission as a trade off:
				// container users should not go really far if this is all they are allowed to do (besides runc will end up giving the mknod permission anyway
                                // https://github.com/opencontainers/runc/blob/ef5e8a5505d6fe022daf016e2535adbda0d89c72/libcontainer/specconv/spec_linux.go#L220-L235)
				device.ContainerEdits.DeviceNodes[i].Permissions = "m"
			}
		}

		cdiDevice := cdi.Device{
			Device: &device,
		}
		if err := cdiDevice.ApplyEdits(spec); err != nil {
			return fmt.Errorf("failed to apply edits for device %q: %v", cdiDevice.GetQualifiedName(), err)
		}
	}

	return m.cdiSpec.ApplyEdits(spec)
}
