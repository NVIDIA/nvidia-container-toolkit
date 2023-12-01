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

package modifier

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestGraphicsModifier(t *testing.T) {
	testCases := []struct {
		description      string
		envmap           map[string]string
		expectedRequired bool
	}{
		{
			description: "empty image does not create modifier",
		},
		{
			description: "devices with no capabilities does not create modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "all",
			},
		},
		{
			description: "devices with no non-graphics does not create modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "compute",
			},
		},
		{
			description: "devices with all capabilities creates modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "all",
			},
			expectedRequired: true,
		},
		{
			description: "devices with graphics capability creates modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "graphics",
			},
			expectedRequired: true,
		},
		{
			description: "devices with compute,graphics capability creates modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "compute,graphics",
			},
			expectedRequired: true,
		},
		{
			description: "devices with display capability creates modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "display",
			},
			expectedRequired: true,
		},
		{
			description: "devices with display,graphics capability creates modifier",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES":     "all",
				"NVIDIA_DRIVER_CAPABILITIES": "display,graphics",
			},
			expectedRequired: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := image.New(
				image.WithEnvMap(tc.envmap),
			)
			required, _ := requiresGraphicsModifier(image)
			require.EqualValues(t, tc.expectedRequired, required)
		})
	}
}
