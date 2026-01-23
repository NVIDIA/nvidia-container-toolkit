/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package cudacompat

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGetCUDACompatElfHeader(t *testing.T) {
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	dataRoot := filepath.Join(moduleRoot, "testdata", "compat")

	testCases := []struct {
		description string
		filename    string
		expected    *compatElfHeader
	}{
		{
			description: "wip",
			filename:    "libcuda.so.575.57.08",
			expected: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
		},
		{
			description: "wip",
			filename:    "libcuda.so.590.44.01",
			expected: &compatElfHeader{
				Format:      1,
				CUDAVersion: "13.1",
				Driver:      []int{535, 550, 570, 575, 580, 590},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			libpath := filepath.Join(dataRoot, tc.filename)

			h, err := GetCUDACompatElfHeader(libpath)
			require.NoError(t, err)

			require.EqualValues(t, tc.expected, h)
		})
	}
}
