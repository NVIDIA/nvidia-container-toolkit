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
	"testing"

	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/stretchr/testify/require"
)

func TestMergeDeviceSpecs(t *testing.T) {
	testCases := []struct {
		description      string
		deviceSpecs      []specs.Device
		mergedDeviceName string
		expectedError    error
		expected         specs.Device
	}{
		{
			description:      "no devices",
			mergedDeviceName: "all",
			expected: specs.Device{
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
			expected: specs.Device{
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
			expected: specs.Device{
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
			expectedError: fmt.Errorf("device %q already exists", "gpu0"),
		},
		{
			description:      "invalid merged device name",
			mergedDeviceName: ".-not-valid",
			expectedError:    fmt.Errorf("invalid device name %q", ".-not-valid"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mergedDevice, err := MergeDeviceSpecs(tc.deviceSpecs, tc.mergedDeviceName)

			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, tc.expected, mergedDevice)
		})
	}
}
