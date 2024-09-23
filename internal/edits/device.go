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

	"golang.org/x/sys/unix"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type device discover.Device

// toEdits converts a discovered device to CDI Container Edits.
func (d device) toEdits(allowAdditionalGIDs bool) (*cdi.ContainerEdits, error) {
	deviceNode, err := d.toSpec()
	if err != nil {
		return nil, err
	}

	var additionalGIDs []uint32
	if allowAdditionalGIDs {
		if requiredGID := d.getRequiredGID(); requiredGID != 0 {
			additionalGIDs = append(additionalGIDs, requiredGID)
		}
	}

	e := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{
			DeviceNodes:    []*specs.DeviceNode{deviceNode},
			AdditionalGIDs: additionalGIDs,
		},
	}
	return &e, nil
}

// toSpec converts a discovered Device to a CDI Spec Device. Note
// that missing info is filled in when edits are applied by querying the Device node.
func (d device) toSpec() (*specs.DeviceNode, error) {
	// The HostPath field was added in the v0.5.0 CDI specification.
	// The cdi package uses strict unmarshalling when loading specs from file causing failures for
	// unexpected fields.
	// Since the behaviour for HostPath == "" and HostPath == Path are equivalent, we clear HostPath
	// if it is equal to Path to ensure compatibility with the widest range of specs.
	hostPath := d.HostPath
	if hostPath == d.Path {
		hostPath = ""
	}

	s := specs.DeviceNode{
		HostPath: hostPath,
		Path:     d.Path,
	}

	return &s, nil
}

// getRequiredGID returns the group id of the device if the device is not world read/writable.
// If the information cannot be extracted or an error occurs, 0 is returned.
func (d device) getRequiredGID() uint32 {
	path := d.HostPath
	if path == "" {
		path = d.Path
	}
	if path == "" {
		return 0
	}

	var stat unix.Stat_t
	if err := unix.Lstat(path, &stat); err != nil {
		return 0
	}
	// This is only supported for char devices
	if stat.Mode&unix.S_IFMT != unix.S_IFCHR {
		return 0
	}

	if permissionsForOther := os.FileMode(stat.Mode).Perm(); permissionsForOther&06 == 0 {
		return stat.Gid
	}
	return 0
}
