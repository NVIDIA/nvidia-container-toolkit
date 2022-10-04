/*
 * Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY Type, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cdi

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

// nvmlDevice wraps an nvml.Device with more functions.
type nvmlDevice struct {
	nvml.Device
}

// nvmlMigDevice allows for specific functions of nvmlDevice to be overridden.
type nvmlMigDevice nvmlDevice

// deviceInfo defines the information the required to construct a Device
type deviceInfo interface {
	GetUUID() (string, error)
	GetDeviceNodes() ([]string, error)
}

var _ deviceInfo = (*nvmlDevice)(nil)
var _ deviceInfo = (*nvmlMigDevice)(nil)

func newGPUDevice(i int, gpu device.Device) (string, nvmlDevice) {
	return fmt.Sprintf("%v", i), nvmlDevice{gpu}
}

func newMigDevice(i int, j int, mig device.MigDevice) (string, nvmlMigDevice) {
	return fmt.Sprintf("%v:%v", i, j), nvmlMigDevice{mig}
}

// GetUUID returns the UUID of the device
func (d nvmlDevice) GetUUID() (string, error) {
	uuid, ret := d.Device.GetUUID()
	if ret != nvml.SUCCESS {
		return "", ret
	}
	return uuid, nil
}

// GetUUID returns the UUID of the device
func (d nvmlMigDevice) GetUUID() (string, error) {
	return nvmlDevice(d).GetUUID()
}

// GetDeviceNodes returns the device node paths for a GPU device
func (d nvmlDevice) GetDeviceNodes() ([]string, error) {
	minor, ret := d.GetMinorNumber()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting GPU device minor number: %v", ret)
	}
	path := fmt.Sprintf("/dev/nvidia%d", minor)

	return []string{path}, nil
}

// GetDeviceNodes returns the device node paths for a MIG device
func (d nvmlMigDevice) GetDeviceNodes() ([]string, error) {
	parent, ret := d.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting parent device: %v", ret)
	}
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

	devicePaths := []string{
		parentPath,
		giCapDevicePath,
		ciCapDevicePath,
	}

	return devicePaths, nil
}
