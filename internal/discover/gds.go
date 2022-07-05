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

type gdsDeviceDiscoverer struct {
	None
	logger  *logrus.Logger
	devices Discover
	mounts  Discover
}

// NewGDSDiscoverer creates a discoverer for GPUDirect Storage devices and mounts.
func NewGDSDiscoverer(logger *logrus.Logger, root string) (Discover, error) {
	devices := &mounts{
		logger:   logger,
		lookup:   lookup.NewCharDeviceLocator(logger, root),
		required: []string{"/dev/nvidia-fs*"},
	}

	udev := &mounts{
		logger:   logger,
		lookup:   lookup.NewDirectoryLocator(logger, root),
		required: []string{"/run/udev"},
	}

	cufile := &mounts{
		logger:   logger,
		lookup:   lookup.NewFileLocator(logger, root),
		required: []string{"/etc/cufile.json"},
	}

	d := gdsDeviceDiscoverer{
		logger:  logger,
		devices: devices,
		mounts:  Merge(udev, cufile),
	}

	return &d, nil
}

// Devices discovers the nvidia-fs device nodes for use with GPUDirect Storage
func (d *gdsDeviceDiscoverer) Devices() ([]Device, error) {
	devicesAsMounts, err := d.devices.Mounts()
	if err != nil {
		d.logger.Debugf("Could not locate GDS devices: %v", err)
		return nil, nil
	}
	var devices []Device
	for _, mount := range devicesAsMounts {
		devices = append(devices, Device(mount))
	}

	return devices, nil
}

// Mounts discovers the required mounts for GPUDirect Storage.
// If no devices are discovered the discovered mounts are empty
func (d *gdsDeviceDiscoverer) Mounts() ([]Mount, error) {
	devices, err := d.Devices()
	if err != nil || len(devices) == 0 {
		d.logger.Debugf("No nvidia-fs devices detected; skipping detection of mounts")
		return nil, nil
	}

	return d.mounts.Mounts()
}
