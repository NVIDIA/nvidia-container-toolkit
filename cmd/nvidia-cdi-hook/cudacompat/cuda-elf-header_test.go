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
			description: "575.57.08",
			filename:    "575.57.08/libcuda.so.575.57.08",
			expected: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
		},
		{
			description: "590.44.01",
			filename:    "590.44.01/libcuda.so.590.44.01",
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

func TestUseCompat(t *testing.T) {
	testCases := []struct {
		description         string
		elfHeader           *compatElfHeader
		compatDriverVersion string
		hostDriverVersion   string
		hostCudaVersion     string
		expected            bool
	}{
		{
			description: "container cuda version greater than host cuda version",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			hostCudaVersion: "12.8",
			expected:        true,
		},
		{
			description: "container cuda version same as host cuda version",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			hostCudaVersion: "12.9",
			expected:        false,
		},
		{
			description: "container cuda version less than host cuda version",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			hostCudaVersion: "12.10",
			expected:        false,
		},
		{
			description: "host driver branch not supported in compat elf header",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			compatDriverVersion: "575.57.08",
			hostDriverVersion:   "590.44.01",
			expected:            false,
		},
		{
			description: "host driver branch supported in compat elf header, host driver branch < compat driver branch",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			compatDriverVersion: "575.57.08",
			hostDriverVersion:   "570.211.01",
			expected:            true,
		},
		{
			description: "host driver branch same as compat driver branch, compat driver > host driver",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			compatDriverVersion: "575.57.08",
			hostDriverVersion:   "575.10.10",
			expected:            true,
		},
		{
			description: "host driver branch same as compat driver branch, compat driver = host driver",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			compatDriverVersion: "575.57.08",
			hostDriverVersion:   "575.57.08",
			expected:            false,
		},
		{
			description: "host driver branch same as compat driver branch, compat driver < host driver",
			elfHeader: &compatElfHeader{
				Format:      1,
				CUDAVersion: "12.9",
				Driver:      []int{535, 550, 560, 565, 570, 575},
				Device:      []int{1, 2, 7, 8, 9, 10, 11, 12, 13, 14},
			},
			compatDriverVersion: "575.57.08",
			hostDriverVersion:   "575.99.99",
			expected:            false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			useCompat := tc.elfHeader.UseCompat(tc.compatDriverVersion, tc.hostDriverVersion, tc.hostCudaVersion)

			require.EqualValues(t, tc.expected, useCompat)
		})
	}
}
