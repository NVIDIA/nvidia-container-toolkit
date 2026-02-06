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

package edits

import (
	"os"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/opencontainers/runc/libcontainer/devices"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type device struct {
	discover.Device
	noAdditionalGIDs bool
}

// toEdits converts a discovered device to CDI Container Edits.
func (d device) toEdits() (*cdi.ContainerEdits, error) {
	deviceNode, err := d.toSpec()
	if err != nil {
		return nil, err
	}

	e := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{
			DeviceNodes:    []*specs.DeviceNode{deviceNode},
			AdditionalGIDs: d.getAdditionalGIDs(deviceNode),
		},
	}
	return &e, nil
}

// toSpec converts a discovered Device to a CDI Spec Device. Note
// that missing info is filled in when edits are applied by querying the Device node.
func (d device) toSpec() (*specs.DeviceNode, error) {
	s := d.fromPathOrDefault()
	// The HostPath field was added in the v0.5.0 CDI specification.
	// The cdi package uses strict unmarshalling when loading specs from file causing failures for
	// unexpected fields.
	// Since the behaviour for HostPath == "" and HostPath == Path are equivalent, we clear HostPath
	// if it is equal to Path to ensure compatibility with the widest range of specs.
	if s.HostPath == d.Path {
		s.HostPath = ""
	}

	return s, nil
}

// fromPathOrDefault attempts to return the returns the information about the
// CDI device from the specified host path.
// If this fails a minimal device is returned so that this information can be
// queried by the container runtime such as containerd.
func (d device) fromPathOrDefault() *specs.DeviceNode {
	dn, err := devices.DeviceFromPath(d.HostPath, "rwm")
	if err != nil {
		return &specs.DeviceNode{
			HostPath: d.HostPath,
			Path:     d.Path,
		}
	}

	return &specs.DeviceNode{
		HostPath:    d.HostPath,
		Path:        d.Path,
		Major:       dn.Major,
		Minor:       dn.Minor,
		FileMode:    &dn.FileMode,
		Permissions: string(dn.Permissions),
		GID:         ptrIfNonZero(dn.Gid),
	}
}

func ptrIfNonZero(id uint32) *uint32 {
	if id == 0 {
		return nil
	}
	return &id
}

// getAdditionalGIDs returns the group id of the device if the device is not world read/writable.
// If the information cannot be extracted or an error occurs, 0 is returned.
func (d *device) getAdditionalGIDs(dn *specs.DeviceNode) []uint32 {
	if d.noAdditionalGIDs {
		return nil
	}
	// Handle the underdefined cases where we do not have enough information to
	// extract the GID for the device OR whether the additional GID is required.
	if dn.GID == nil {
		return nil
	}
	if dn.FileMode == nil {
		return nil
	}
	if dn.FileMode.Type() != os.ModeCharDevice {
		return nil
	}

	if permissionsForOther := dn.FileMode.Perm(); permissionsForOther&06 != 0 {
		return []uint32{*dn.GID}
	}

	return nil
}
