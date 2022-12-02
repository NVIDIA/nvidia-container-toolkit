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

package generate

import (
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/sirupsen/logrus"
)

// deviceDiscoverer defines a discoverer for device nodes
type deviceDiscoverer struct {
	logger          *logrus.Logger
	root            string
	deviceNodePaths []string
}

var _ discover.Discover = (*deviceDiscoverer)(nil)

// Devices returns the device nodes for the full GPU.
func (d *deviceDiscoverer) Devices() ([]discover.Device, error) {
	var deviceNodes []discover.Device
	for _, dn := range d.deviceNodePaths {
		deviceNode := discover.Device{
			HostPath: filepath.Join(d.root, dn),
			Path:     dn,
		}
		deviceNodes = append(deviceNodes, deviceNode)
	}

	return deviceNodes, nil
}

// Hooks returns no hooks for a device discoverer
func (d *deviceDiscoverer) Hooks() ([]discover.Hook, error) {
	return nil, nil
}

// Mounts returns no mounts for a device discoverer
func (d *deviceDiscoverer) Mounts() ([]discover.Mount, error) {
	return nil, nil
}
