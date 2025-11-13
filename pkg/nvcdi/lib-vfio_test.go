/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package nvcdi

import (
	"bytes"
	"testing"

	"github.com/NVIDIA/go-nvlib/pkg/nvpci"
	"github.com/stretchr/testify/require"
)

func TestModeVfio(t *testing.T) {
	testCases := []struct {
		description   string
		pcilib        *nvpci.InterfaceMock
		ids           []string
		expectedError error
		expectedSpec  string
	}{
		{
			description: "get all specs single device",
			pcilib: &nvpci.InterfaceMock{
				GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
					devices := []*nvpci.NvidiaPCIDevice{
						{
							Driver:     "vfio-pci",
							IommuGroup: 5,
						},
					}
					return devices, nil
				},
			},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: nvidia.com/pgpu
devices:
    - name: "0"
      containerEdits:
        deviceNodes:
            - path: /dev/vfio/5
              hostPath: /dev/vfio/5
containerEdits:
    env:
        - NVIDIA_VISIBLE_DEVICES=void
    deviceNodes:
        - path: /dev/vfio/vfio
          hostPath: /dev/vfio/vfio
`,
		},
		{
			description: "get single device spec by index",
			pcilib: &nvpci.InterfaceMock{
				GetGPUByIndexFunc: func(n int) (*nvpci.NvidiaPCIDevice, error) {
					devices := []*nvpci.NvidiaPCIDevice{
						{
							Driver:     "vfio-pci",
							IommuGroup: 45,
						},
						{
							Driver:     "vfio-pci",
							IommuGroup: 5,
						},
					}
					return devices[n], nil
				},
			},
			ids: []string{"1"},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: nvidia.com/pgpu
devices:
    - name: "1"
      containerEdits:
        deviceNodes:
            - path: /dev/vfio/5
              hostPath: /dev/vfio/5
containerEdits:
    env:
        - NVIDIA_VISIBLE_DEVICES=void
    deviceNodes:
        - path: /dev/vfio/vfio
          hostPath: /dev/vfio/vfio
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			lib, err := New(
				WithMode(ModeVfio),
				WithPCILib(tc.pcilib),
			)
			require.NoError(t, err)

			spec, err := lib.GetSpec(tc.ids...)
			require.EqualValues(t, tc.expectedError, err)

			var output bytes.Buffer

			_, err = spec.WriteTo(&output)
			require.NoError(t, err)

			require.Equal(t, tc.expectedSpec, output.String())
		})
	}

}
