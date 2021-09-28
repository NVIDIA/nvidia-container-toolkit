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

package discover

// DevicePath is a path in /dev associated with a device
type DevicePath string

// ProcPath is a path in /proc associated with a devices
type ProcPath string

// PCIBusID is the ID on the PCI bus of a device
type PCIBusID string

// DeviceNode represents a device on the file system
type DeviceNode struct {
	Path  DevicePath
	Major int
	Minor int
}

// Device represents a discovered device including identifiers (Index, UUID, PCI bus ID)
// for selection and paths in /dev and /proc associated with the device
type Device struct {
	Index       string
	UUID        string
	PCIBusID    PCIBusID
	DeviceNodes []DeviceNode
	ProcPaths   []ProcPath
}

// Mount represents a discovered mount. This includes a set of labels
// for selection and the mount path
type Mount struct {
	Path   string
	Labels map[string]string
}

// Hook represents a discovered hook
type Hook struct {
	Path     string
	Args     []string
	HookName string
	Labels   map[string]string
}

// Discover defines an interface for discovering the devices and mounts available on a system
type Discover interface {
	Devices() ([]Device, error)
	Mounts() ([]Mount, error)
	Hooks() ([]Hook, error)
}
