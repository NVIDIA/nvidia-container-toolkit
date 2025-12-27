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

package oci

import (
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

type DeviceResolver interface {
	GlobDevices(pattern string) ([]string, error)
	DevicePathToRule(path string) (*specs.LinuxDeviceCgroup, error)
}

type RealDeviceResolver struct {
	devRoot string
}

func NewRealDeviceResolver(devRoot string) *RealDeviceResolver {
	return &RealDeviceResolver{devRoot: devRoot}
}

func (r *RealDeviceResolver) GlobDevices(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(r.devRoot, pattern))
}

func (r *RealDeviceResolver) DevicePathToRule(path string) (*specs.LinuxDeviceCgroup, error) {
	var stat unix.Stat_t
	if err := unix.Stat(path, &stat); err != nil {
		return nil, err
	}

	if stat.Mode&unix.S_IFCHR == 0 {
		return nil, fmt.Errorf("%s is not a character device", path)
	}

	major := int64(unix.Major(stat.Rdev))
	minor := int64(unix.Minor(stat.Rdev))

	return &specs.LinuxDeviceCgroup{
		Allow:  true,
		Type:   "c",
		Major:  &major,
		Minor:  &minor,
		Access: "rwm",
	}, nil
}
