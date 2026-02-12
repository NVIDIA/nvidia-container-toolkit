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
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencontainers/cgroups/devices/config"
)

//go:generate moq -rm -fmt=goimports -out devices_mock.go . Interface
type Interface interface {
	DeviceFromPath(string, string) (*Device, error)
	AssertCharDevice(string) error
}

type testDefaults struct{}

var _ Interface = (*testDefaults)(nil)

func SetAllForTest() func() {
	d := &testDefaults{}
	return SetInterfaceForTests(d)
}

func SetInterfaceForTests(m Interface) func() {
	if m == nil {
		return SetAllForTest()
	}
	funcs := []func(){
		SetDeviceFromPathForTest(m.DeviceFromPath),
		SetAssertCharDeviceForTest(m.AssertCharDevice),
	}
	return func() {
		for _, f := range funcs {
			f()
		}
	}
}

func SetDeviceFromPathForTest(testFunc func(string, string) (*Device, error)) func() {
	current := deviceFromPathStub
	deviceFromPathStub = testFunc
	return func() {
		deviceFromPathStub = current
	}
}

func SetAssertCharDeviceForTest(testFunc func(string) error) func() {
	current := assertCharDeviceStub
	assertCharDeviceStub = testFunc
	return func() {
		assertCharDeviceStub = current
	}
}

type testDevice struct {
	Device
}

func (t *testDevice) load() error {
	deviceFile, err := os.Open(t.Path)
	if err != nil {
		return err
	}
	defer deviceFile.Close()

	decoder := json.NewDecoder(deviceFile)
	return decoder.Decode(&t)
}

func (t *testDefaults) DeviceFromPath(path string, permissions string) (*Device, error) {
	device := testDevice{
		Device: Device{
			Path: path,
			Rule: config.Rule{
				Permissions: config.Permissions(permissions),
			},
		},
	}

	if err := device.load(); err != nil {
		return nil, err
	}

	return &device.Device, nil
}

func (t *testDefaults) AssertCharDevice(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error getting info for %v: %v", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("specified path '%v' is a directory", path)
	}

	return nil
}
