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

func TestDeduplicate(t *testing.T) {
	testCases := []struct {
		description   string
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description: "nil spec",
		},
		{
			description: "duplicate deviceNode is removed",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{
							Path: "/dev/gpu0",
						},
						{
							Path: "/dev/gpu0",
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{
							Path: "/dev/gpu0",
						},
					},
				},
			},
		},
		{
			description: "duplicate deviceNode is remved from device edits",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/gpu0",
								},
								{
									Path: "/dev/gpu0",
								},
							},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/gpu0",
								},
							},
						},
					},
				},
			},
		},
		{
			description: "duplicate hook is removed",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-ctk",
							Args:     []string{"nvidia-ctk", "hook", "chmod", "--mode", "755", "--path", "/dev/dri"},
						},
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-ctk",
							Args:     []string{"nvidia-ctk", "hook", "chmod", "--mode", "755", "--path", "/dev/dri"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-ctk",
							Args:     []string{"nvidia-ctk", "hook", "chmod", "--mode", "755", "--path", "/dev/dri"},
						},
					},
				},
			},
		},
		{
			description: "duplicate mount is removed",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Mounts: []*specs.Mount{
								{
									HostPath:      "/host/mount2",
									ContainerPath: "/mount2",
								},
								{
									HostPath:      "/host/mount2",
									ContainerPath: "/mount2",
								},
								{
									HostPath:      "/host/mount1",
									ContainerPath: "/mount1",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/mount1",
							ContainerPath: "/mount1",
							Options:       []string{"bind", "ro"},
							Type:          "tmpfs",
						},
						{
							HostPath:      "/host/mount1",
							ContainerPath: "/mount1",
							Options:       []string{"bind", "ro"},
							Type:          "tmpfs",
						},
						{
							HostPath:      "/host/mount1",
							ContainerPath: "/mount1",
							Options:       []string{"bind", "ro"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Mounts: []*specs.Mount{
								{
									HostPath:      "/host/mount2",
									ContainerPath: "/mount2",
								},
								{
									HostPath:      "/host/mount1",
									ContainerPath: "/mount1",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/mount1",
							ContainerPath: "/mount1",
							Options:       []string{"bind", "ro"},
							Type:          "tmpfs",
						},
						{
							HostPath:      "/host/mount1",
							ContainerPath: "/mount1",
							Options:       []string{"bind", "ro"},
						},
					},
				},
			},
		},
		{
			description: "duplicate env is removed",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Env: []string{"ENV1=VAL1", "ENV1=VAL1", "ENV2=ONE_VALUE", "ENV2=ANOTHER_VALUE"},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"ENV1=VAL1", "ENV1=VAL1", "ENV2=ONE_VALUE", "ENV2=ANOTHER_VALUE"},
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Env: []string{"ENV1=VAL1", "ENV2=ONE_VALUE", "ENV2=ANOTHER_VALUE"},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"ENV1=VAL1", "ENV2=ONE_VALUE", "ENV2=ANOTHER_VALUE"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := dedupe{}

			err := d.Transform(tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}
}
