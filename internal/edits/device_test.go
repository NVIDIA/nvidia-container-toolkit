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
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

func TestDeviceToSpec(t *testing.T) {
	testCases := []struct {
		device   discover.Device
		expected *specs.DeviceNode
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
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			spec, err := device(tc.device).toSpec()
			require.NoError(t, err)
			require.EqualValues(t, tc.expected, spec)
		})
	}
}
