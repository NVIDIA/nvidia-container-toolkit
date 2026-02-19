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

package edits

import (
	"fmt"
	"os"
	"testing"

	"github.com/opencontainers/cgroups/devices/config"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test/to"
)

func TestDeviceToSpec(t *testing.T) {
	testCases := []struct {
		description string
		device      discover.Device
		deviceslib  devices.Interface
		expected    *specs.DeviceNode
	}{
		{
			device: discover.Device{
				Path: "/foo",
			},
			expected: &specs.DeviceNode{
				Path: "/foo",
			},
		},
		{
			device: discover.Device{
				Path:     "/foo",
				HostPath: "/foo",
			},
			expected: &specs.DeviceNode{
				Path: "/foo",
			},
		},
		{
			device: discover.Device{
				Path:     "/foo",
				HostPath: "/not/foo",
			},
			expected: &specs.DeviceNode{
				Path:     "/foo",
				HostPath: "/not/foo",
			},
		},
		{
			description: "device with device properties",
			device: discover.Device{
				Path: "/foo",
			},
			deviceslib: &devices.InterfaceMock{
				DeviceFromPathFunc: func(path, permissions string) (*devices.Device, error) {
					if path != "/foo" {
						return nil, fmt.Errorf("not found %v", path)
					}
					cd := &config.Device{
						Rule: config.Rule{
							Major:       100,
							Minor:       200,
							Permissions: config.Permissions("w"),
						},
						Uid: 11,
						Gid: 44,
					}

					return (*devices.Device)(cd), nil
				},
			},
			expected: &specs.DeviceNode{
				Path:        "/foo",
				HostPath:    "",
				Permissions: "w",
				Major:       100,
				Minor:       200,
				GID:         ptrIfNonZero[uint32](44),
			},
		},
		{
			description: "device with additional GIDs",
			device: discover.Device{
				Path: "/foo",
			},
			deviceslib: &devices.InterfaceMock{
				DeviceFromPathFunc: func(path, permissions string) (*devices.Device, error) {
					if path != "/foo" {
						return nil, fmt.Errorf("not found %v", path)
					}
					cd := &config.Device{
						Rule: config.Rule{
							Major:       100,
							Minor:       200,
							Permissions: config.Permissions("w"),
						},
						FileMode: 0660 | os.ModeCharDevice,
						Uid:      11,
						Gid:      44,
					}

					return (*devices.Device)(cd), nil
				},
			},
			expected: &specs.DeviceNode{
				Path:        "/foo",
				HostPath:    "",
				Permissions: "w",
				Major:       100,
				Minor:       200,
				FileMode:    to.Ptr(0660 | os.ModeCharDevice),
				GID:         ptrIfNonZero[uint32](44),
			},
		},
	}

	for _, tc := range testCases {
		f := factory{}
		t.Run(tc.description, func(t *testing.T) {
			defer devices.SetInterfaceForTests(tc.deviceslib)()
			spec, err := f.device(tc.device).toSpec()
			require.NoError(t, err)
			require.EqualValues(t, tc.expected, spec)
		})
	}
}

func TestGetAdditionalGIDs(t *testing.T) {
	testCases := []struct {
		description            string
		device                 *device
		deviceNode             *specs.DeviceNode
		expectedAdditionalGIDs []uint32
	}{
		{
			description: "feature disabled",
			device:      &device{noAdditionalGIDs: true},
		},
		{
			description: "device node has no GID",
			device:      &device{},
		},
		{
			description: "device node has zero GID",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID: to.Ptr[uint32](0),
			},
		},
		{
			description: "filemode not specified",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID: to.Ptr[uint32](1),
			},
		},
		{
			description: "device node is not a character device",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID:      to.Ptr[uint32](1),
				FileMode: to.Ptr(0666 | os.ModeSymlink),
			},
		},
		{
			description: "character device is world read-writeable",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID:      to.Ptr[uint32](1),
				FileMode: to.Ptr(0666 | os.ModeCharDevice),
			},
		},
		{
			description: "character device is only world readable",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID:      to.Ptr[uint32](1),
				FileMode: to.Ptr(0664 | os.ModeCharDevice),
			},
			expectedAdditionalGIDs: []uint32{1},
		},
		{
			description: "character device is only world writeable",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID:      to.Ptr[uint32](1),
				FileMode: to.Ptr(0662 | os.ModeCharDevice),
			},
			expectedAdditionalGIDs: []uint32{1},
		},
		{
			description: "character device is not world read-writeable",
			device:      &device{},
			deviceNode: &specs.DeviceNode{
				GID:      to.Ptr[uint32](1),
				FileMode: to.Ptr(0660 | os.ModeCharDevice),
			},
			expectedAdditionalGIDs: []uint32{1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			additionalGIDs := tc.device.getAdditionalGIDs(tc.deviceNode)

			require.EqualValues(t, tc.expectedAdditionalGIDs, additionalGIDs)
		})
	}
}
