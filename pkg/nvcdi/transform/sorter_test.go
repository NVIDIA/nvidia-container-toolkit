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

package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"
)

func TestSortSpec(t *testing.T) {
	testCases := []struct {
		description string
		spec        *specs.Spec
		expected    *specs.Spec
	}{
		{
			description: "sort sorts devices by name and device edits",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device2",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/device2/dev0",
								},
								{
									Path: "/dev/device2/dev1",
								},
							},
							Mounts: []*specs.Mount{
								{
									ContainerPath: "/lib/device2/mount1",
								},
								{
									ContainerPath: "/lib/device2/mount0",
								},
							},
						},
					},
					{
						Name: "device1",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/device1/dev1",
								},
								{
									Path: "/dev/device1/dev0",
								},
							},
							Mounts: []*specs.Mount{
								{
									ContainerPath: "/lib/device1/mount0",
								},
								{
									ContainerPath: "/lib/device1/mount1",
								},
							},
						},
					},
				},
			},
			expected: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device1",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/device1/dev0",
								},
								{
									Path: "/dev/device1/dev1",
								},
							},
							Mounts: []*specs.Mount{
								{
									ContainerPath: "/lib/device1/mount0",
								},
								{
									ContainerPath: "/lib/device1/mount1",
								},
							},
						},
					},
					{
						Name: "device2",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/device2/dev0",
								},
								{
									Path: "/dev/device2/dev1",
								},
							},
							Mounts: []*specs.Mount{
								{
									ContainerPath: "/lib/device2/mount0",
								},
								{
									ContainerPath: "/lib/device2/mount1",
								},
							},
						},
					},
				},
			},
		},
		{
			description: "sort sorts common edits",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{
							Path: "/dev/device2/dev0",
						},
						{
							Path: "/dev/device2/dev1",
						},
						{
							Path: "/dev/device1/dev1",
						},
						{
							Path: "/dev/device1/dev0",
						},
					},
					Mounts: []*specs.Mount{
						{
							ContainerPath: "/lib/device2/mount1",
						},
						{
							ContainerPath: "/lib/device2/mount0",
						},
						{
							ContainerPath: "/lib/device1/mount0",
						},
						{
							ContainerPath: "/lib/device1/mount1",
						},
					},
				},
			},
			expected: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{
							Path: "/dev/device1/dev0",
						},
						{
							Path: "/dev/device1/dev1",
						},
						{
							Path: "/dev/device2/dev0",
						},
						{
							Path: "/dev/device2/dev1",
						},
					},
					Mounts: []*specs.Mount{
						{
							ContainerPath: "/lib/device1/mount0",
						},
						{
							ContainerPath: "/lib/device1/mount1",
						},
						{
							ContainerPath: "/lib/device2/mount0",
						},
						{
							ContainerPath: "/lib/device2/mount1",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			s := sorter{}
			err := s.Transform(tc.spec)
			require.NoError(t, err)

			require.EqualValues(t, tc.expected, tc.spec)
		})
	}
}

func TestSortDeviceNodes(t *testing.T) {
	testCases := []struct {
		description         string
		deviceNodes         []*specs.DeviceNode
		expectedDeviceNodes []*specs.DeviceNode
	}{
		{
			description: "sorted remains sorted",
			deviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia0",
				},
				{
					Path: "/dev/nvidia1",
				},
			},
			expectedDeviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia0",
				},
				{
					Path: "/dev/nvidia1",
				},
			},
		},
		{
			description: "unsorted gets sorted",
			deviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia1",
				},
				{
					Path: "/dev/nvidia0",
				},
			},
			expectedDeviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia0",
				},
				{
					Path: "/dev/nvidia1",
				},
			},
		},
		{
			description: "shorter paths are first",
			deviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia0/another",
				},
				{
					Path: "/dev/nvidia0",
				},
			},
			expectedDeviceNodes: []*specs.DeviceNode{
				{
					Path: "/dev/nvidia0",
				},
				{
					Path: "/dev/nvidia0/another",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			s := sorter{}
			sorted := s.sortDeviceNodes(tc.deviceNodes)

			require.EqualValues(t, tc.expectedDeviceNodes, sorted)
		})
	}
}

func TestStortMounts(t *testing.T) {
	testCases := []struct {
		description    string
		mounts         []*specs.Mount
		expectedMounts []*specs.Mount
	}{
		{
			description: "sorted remains sorted",
			mounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia0",
				},
				{
					ContainerPath: "/lib/nvidia1",
				},
			},
			expectedMounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia0",
				},
				{
					ContainerPath: "/lib/nvidia1",
				},
			},
		},
		{
			description: "unsorted gets sorted",
			mounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia1",
				},
				{
					ContainerPath: "/lib/nvidia0",
				},
			},
			expectedMounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia0",
				},
				{
					ContainerPath: "/lib/nvidia1",
				},
			},
		},
		{
			description: "shorter paths are first",
			mounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia0/another",
				},
				{
					ContainerPath: "/lib/nvidia0",
				},
			},
			expectedMounts: []*specs.Mount{
				{
					ContainerPath: "/lib/nvidia0",
				},
				{
					ContainerPath: "/lib/nvidia0/another",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			s := sorter{}
			sorted := s.sortMounts(tc.mounts)

			require.EqualValues(t, tc.expectedMounts, sorted)
		})
	}
}
