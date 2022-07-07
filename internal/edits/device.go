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
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/container-orchestrated-devices/container-device-interface/specs-go"

	"github.com/opencontainers/runc/libcontainer/devices"
)

type device discover.Device

// toEdits converts a discovered device to CDI Container Edits.
func (d device) toEdits() (*cdi.ContainerEdits, error) {
	deviceNode, err := d.toSpec()
	if err != nil {
		return nil, err
	}

	e := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{
			DeviceNodes: []*specs.DeviceNode{deviceNode},
		},
	}
	return &e, nil
}

// toSpec converts a discovered Device to a CDI Spec Device. Note
// that missing info is filled in when edits are applied by querying the Device node.
func (d device) toSpec() (*specs.DeviceNode, error) {
	// NOTE: This mirrors what cri-o does.
	// https://github.com/cri-o/cri-o/blob/ca3bb80a3dda0440659fcf8da8ed6f23211de94e/internal/config/device/device.go#L93
	// This can be removed once https://github.com/container-orchestrated-devices/container-device-interface/issues/72 is addressed
	dev, err := devices.DeviceFromPath(d.HostPath, "rwm")
	if err != nil {
		return nil, fmt.Errorf("failed to query device node %v: %v", d.HostPath, err)
	}
	s := specs.DeviceNode{
		Path:     d.Path,
		Type:     string(dev.Type),
		Major:    dev.Major,
		Minor:    dev.Minor,
		FileMode: &dev.FileMode,
		UID:      &dev.Uid,
		GID:      &dev.Gid,
	}

	return &s, nil
}
