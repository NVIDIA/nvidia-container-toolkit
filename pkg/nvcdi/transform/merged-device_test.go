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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"
)

func TestMergeDeviceSpecs(t *testing.T) {
	testCases := []struct {
		description      string
		deviceSpecs      []specs.Device
		mergedDeviceName string
		createError      error
		expectedError    error
		expected         *specs.Device
	}{
		{
			description:      "no devices",
			mergedDeviceName: "all",
			expected: &specs.Device{
				Name: "all",
			},
		},
		{
			description:      "one device",
			mergedDeviceName: "all",
			deviceSpecs: []specs.Device{
				{
					Name: "gpu0",
					ContainerEdits: specs.ContainerEdits{
						Env: []string{"GPU=0"},
					},
				},
			},
			expected: &specs.Device{
				Name: "all",
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"GPU=0"},
				},
			},
		},
		{
			description:      "two devices",
			mergedDeviceName: "all",
			deviceSpecs: []specs.Device{
				{
					Name: "gpu0",
					ContainerEdits: specs.ContainerEdits{
						Env: []string{"GPU=0"},
					},
				},
				{
					Name: "gpu1",
					ContainerEdits: specs.ContainerEdits{
						Env: []string{"GPU=1"},
					},
				},
			},
			expected: &specs.Device{
				Name: "all",
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"GPU=0", "GPU=1"},
				},
			},
		},
		{
			description:      "has merged device",
			mergedDeviceName: "gpu0",
			deviceSpecs: []specs.Device{
				{
					Name: "gpu0",
					ContainerEdits: specs.ContainerEdits{
						Env: []string{"GPU=0"},
					},
				},
			},
		},
		{
			description:      "invalid merged device name",
			mergedDeviceName: ".-not-valid",
			createError:      fmt.Errorf("invalid device name %q", ".-not-valid"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := NewMergedDevice(
				WithName(tc.mergedDeviceName),
			)
			if tc.createError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			device, err := mergeDeviceSpecs(tc.deviceSpecs, tc.mergedDeviceName)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.EqualValues(t, tc.expected, device)
		})
	}
}

func TestMergedDevice(t *testing.T) {
	testCases := []struct {
		description   string
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description: "duplicate hooks are removed",
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
				},
			},
			expectedSpec: &specs.Spec{
				Devices: []specs.Device{
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
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m, err := NewMergedDevice()
			require.NoError(t, err)

			err = m.Transform(tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}
}
