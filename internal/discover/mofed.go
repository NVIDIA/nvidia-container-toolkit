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

package discover

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

type mofedDeviceDiscoverer mounts

// NewMOFEDDiscoverer creates a discoverer for MOFED devices.
func NewMOFEDDiscoverer(logger *logrus.Logger, root string) (Discover, error) {
	devices := &mofedDeviceDiscoverer{
		logger: logger,
		lookup: lookup.NewCharDeviceLocator(logger, root),
		required: []string{
			"/dev/infiniband/uverbs*",
			"/dev/infiniband/rdma_cm",
		},
	}

	return devices, nil
}

// Devices discovers the uverbs* and rdma_cm device nodes for use with GPUDirect Storage and the MOFED stack.
func (d *mofedDeviceDiscoverer) Devices() ([]Device, error) {
	devicesAsMounts, err := (*mounts)(d).Mounts()
	if err != nil {
		d.logger.Debugf("Could not locate MOFED devices: %v", err)
		return nil, nil
	}
	var devices []Device
	for _, mount := range devicesAsMounts {
		devices = append(devices, Device(mount))
	}

	return devices, nil
}
