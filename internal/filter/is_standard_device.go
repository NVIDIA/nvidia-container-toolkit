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

import "github.com/NVIDIA/nvidia-container-toolkit/internal/discover"

type standardDevice struct{}

// StandardDevice returns a selector for regular (non-control) devices
func StandardDevice() Selector {
	return &standardDevice{}
}

// Selected returns true for a standard device and false for controll devices. A regular device
// is expected to have an index, uuid, and PCI bus ID.
func (s standardDevice) Selected(device discover.Device) bool {
	if device.Index == "" {
		return false
	}
	if device.PCIBusID == "" {
		return false
	}
	if device.UUID == "" {
		return false
	}
	return true
}
