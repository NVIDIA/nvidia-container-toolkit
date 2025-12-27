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

package oci

import (
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// MockDeviceResolver for testing
type MockDeviceResolver struct {
	devices map[string]struct{ major, minor int64 }
}

func NewMockDeviceResolver() *MockDeviceResolver {
	return &MockDeviceResolver{
		devices: map[string]struct{ major, minor int64 }{
			"nvidia0":          {195, 0},
			"nvidia1":          {195, 1},
			"nvidia2":          {195, 2},
			"nvidiactl":        {195, 255},
			"nvidia-modeset":   {195, 254},
			"nvidia-uvm":       {236, 0},
			"nvidia-uvm-tools": {236, 1},
		},
	}
}

func (r *MockDeviceResolver) GlobDevices(pattern string) ([]string, error) {
	var matches []string
	for name := range r.devices {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

func (r *MockDeviceResolver) DevicePathToRule(path string) (*specs.LinuxDeviceCgroup, error) {
	base := filepath.Base(path)
	dev, ok := r.devices[base]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", path)
	}

	major := dev.major
	minor := dev.minor

	return &specs.LinuxDeviceCgroup{
		Allow:  true,
		Type:   "c",
		Major:  &major,
		Minor:  &minor,
		Access: "rwm",
	}, nil
}
