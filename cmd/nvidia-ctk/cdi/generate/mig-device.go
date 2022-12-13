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
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

// NewMigDeviceDiscoverer creates a discoverer for the specified mig device and its parent.
func NewMigDeviceDiscoverer(logger *logrus.Logger, root string, parent device.Device, d device.MigDevice) (discover.Discover, error) {
	minor, ret := parent.GetMinorNumber()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting GPU device minor number: %v", ret)
	}
	parentPath := fmt.Sprintf("/dev/nvidia%d", minor)

	migCaps, err := nvcaps.NewMigCaps()
	if err != nil {
		return nil, fmt.Errorf("error getting MIG capability device paths: %v", err)
	}

	gi, ret := d.GetGpuInstanceId()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting GPU Instance ID: %v", ret)
	}

	ci, ret := d.GetComputeInstanceId()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting Compute Instance ID: %v", ret)
	}

	giCap := nvcaps.NewGPUInstanceCap(minor, gi)
	giCapDevicePath, err := migCaps.GetCapDevicePath(giCap)
	if err != nil {
		return nil, fmt.Errorf("failed to get GI cap device path: %v", err)
	}

	ciCap := nvcaps.NewComputeInstanceCap(minor, gi, ci)
	ciCapDevicePath, err := migCaps.GetCapDevicePath(ciCap)
	if err != nil {
		return nil, fmt.Errorf("failed to get CI cap device path: %v", err)
	}

	deviceNodes := discover.NewCharDeviceDiscoverer(
		logger,
		[]string{
			parentPath,
			giCapDevicePath,
			ciCapDevicePath,
		},
		root,
	)

	return deviceNodes, nil
}
