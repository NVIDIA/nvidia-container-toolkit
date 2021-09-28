/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package filter

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type devicesByID map[string]struct{}

var _ Selector = (*devicesByID)(nil)

// NewDeviceSelector creates a selector for devices based on the specified IDs.
func NewDeviceSelector(ids ...string) Selector {
	deviceIDs := make(devicesByID)

	for _, id := range ids {
		deviceIDs[id] = struct{}{}
	}

	return deviceIDs
}

// Selected checks whether a specific device is included in the set of devicesIDs
// The device is checked by UUID, Index, and PCIBusID and if any of these match
// the device is considered selected.
func (d devicesByID) Selected(device discover.Device) bool {
	var exists bool

	_, exists = d[device.UUID]
	if exists {
		return true
	}

	_, exists = d[device.Index]
	if exists {
		return true
	}

	_, exists = d[device.PCIBusID.String()]
	if exists {
		return true
	}

	return false
}
