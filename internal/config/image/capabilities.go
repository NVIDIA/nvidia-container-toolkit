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

package image

// DriverCapability represents the possible values of NVIDIA_DRIVER_CAPABILITIES
type DriverCapability string

// Constants for the supported driver capabilities
const (
	DriverCapabilityAll      DriverCapability = "all"
	DriverCapabilityCompat32 DriverCapability = "compat32"
	DriverCapabilityCompute  DriverCapability = "compute"
	DriverCapabilityDisplay  DriverCapability = "display"
	DriverCapabilityGraphics DriverCapability = "graphics"
	DriverCapabilityNgx      DriverCapability = "ngx"
	DriverCapabilityUtility  DriverCapability = "utility"
	DriverCapabilityVideo    DriverCapability = "video"
)

// DriverCapabilities represents the NVIDIA_DRIVER_CAPABILITIES set for the specified image.
type DriverCapabilities map[DriverCapability]bool

// Has check whether the specified capability is selected.
func (c DriverCapabilities) Has(capability DriverCapability) bool {
	if c[DriverCapabilityAll] {
		return true
	}
	return c[capability]
}

// Any checks whether any of the specified capabilites are set
func (c DriverCapabilities) Any(capabilities ...DriverCapability) bool {
	for _, cap := range capabilities {
		if c.Has(cap) {
			return true
		}
	}

	return false
}
