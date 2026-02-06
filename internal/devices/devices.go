/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package devices

import (
	"fmt"
	"os"

	"github.com/opencontainers/cgroups/devices/config"
	"github.com/opencontainers/runc/libcontainer/devices"
)

type Device config.Device

// DeviceFromPath is a wrapper for libcontainer/devices.DeviceFromPath.
// It allows for overriding functionality during tests.
func DeviceFromPath(path string, permissions string) (*Device, error) {
	return deviceFromPathStub(path, permissions)
}

var deviceFromPathStub = func(path string, permissions string) (*Device, error) {
	d, err := devices.DeviceFromPath(path, permissions)
	return (*Device)(d), err
}

// AssertCharDevice checks whether the specified path is a char device and returns an error if this is not the case.
func AssertCharDevice(path string) error {
	return assertCharDeviceStub(path)
}

var assertCharDeviceStub = func(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("error getting info: %v", err)
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("%v is not a char device", path)
	}
	return nil
}
