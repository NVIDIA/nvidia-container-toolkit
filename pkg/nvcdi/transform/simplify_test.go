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

func TestSimplify(t *testing.T) {
	testCases := []struct {
		description   string
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description: "nil spec is a no-op",
		},
		{
			description:  "empty spec is simplified",
			spec:         &specs.Spec{},
			expectedSpec: &specs.Spec{},
		},
		{
			description: "simplify does not allow empty device",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Env: []string{"FOO=BAR"},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"FOO=BAR"},
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Env: []string{"FOO=BAR"},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"FOO=BAR"},
				},
			},
		},
		{
			description: "simplify removes common entities",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "device0",
						ContainerEdits: specs.ContainerEdits{
							Env: []string{"FOO=BAR"},
							DeviceNodes: []*specs.DeviceNode{
								{
									Path: "/dev/gpu0",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"FOO=BAR"},
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
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"FOO=BAR"},
				},
			},
		},
		{
			description: "simplify removes hooks",
			spec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "gpu0",
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
					{
						Name: "gpu1",
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
					{
						Name: "all",
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
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
					{
						Name: "gpu0",
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
					{
						Name: "gpu1",
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
					{
						Name: "all",
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
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			s := simplify{}

			err := s.Transform(tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedSpec, tc.spec)

		})
	}
}
