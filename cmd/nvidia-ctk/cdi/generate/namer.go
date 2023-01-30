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

	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

type deviceNamer interface {
	GetDeviceName(int, device.Device) (string, error)
	GetMigDeviceName(int, int, device.MigDevice) (string, error)
}

const (
	deviceNameStrategyIndex     = "index"
	deviceNameStrategyTypeIndex = "type-index"
	deviceNameStrategyUUID      = "uuid"
)

type deviceNameIndex struct {
	gpuPrefix string
	migPrefix string
}
type deviceNameUUID struct{}

// newDeviceNamer creates a Device Namer based on the supplied strategy.
// This namer can be used to construct the names for MIG and GPU devices when generating the CDI spec.
func newDeviceNamer(strategy string) (deviceNamer, error) {
	switch strategy {
	case deviceNameStrategyIndex:
		return deviceNameIndex{}, nil
	case deviceNameStrategyTypeIndex:
		return deviceNameIndex{gpuPrefix: "gpu", migPrefix: "mig"}, nil
	case deviceNameStrategyUUID:
		return deviceNameUUID{}, nil
	}

	return nil, fmt.Errorf("invalid device name strategy: %v", strategy)
}

// GetDeviceName returns the name for the specified device based on the naming strategy
func (s deviceNameIndex) GetDeviceName(i int, d device.Device) (string, error) {
	return fmt.Sprintf("%s%d", s.gpuPrefix, i), nil
}

// GetMigDeviceName returns the name for the specified device based on the naming strategy
func (s deviceNameIndex) GetMigDeviceName(i int, j int, d device.MigDevice) (string, error) {
	return fmt.Sprintf("%s%d:%d", s.migPrefix, i, j), nil
}

// GetDeviceName returns the name for the specified device based on the naming strategy
func (s deviceNameUUID) GetDeviceName(i int, d device.Device) (string, error) {
	uuid, ret := d.GetUUID()
	if ret != nvml.SUCCESS {
		return "", fmt.Errorf("failed to get device UUID: %v", ret)
	}
	return uuid, nil
}

// GetMigDeviceName returns the name for the specified device based on the naming strategy
func (s deviceNameUUID) GetMigDeviceName(i int, j int, d device.MigDevice) (string, error) {
	uuid, ret := d.GetUUID()
	if ret != nvml.SUCCESS {
		return "", fmt.Errorf("failed to get device UUID: %v", ret)
	}
	return uuid, nil
}
