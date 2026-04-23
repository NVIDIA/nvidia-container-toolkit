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

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		description string
		a           string
		b           string
		expected    int
	}{
		{
			description: "empty",
			expected:    0,
		},
		{
			description: "less than",
			a:           "1.2.3",
			b:           "2.4.5",
			expected:    -1,
		},
		{
			description: "equal",
			a:           "1.1.1",
			b:           "1.1.1",
			expected:    0,
		},
		{
			description: "equal with leading zeros in version string",
			a:           "1.1.1",
			b:           "1.01.1",
			expected:    0,
		},
		{
			description: "greater than",
			a:           "2.4.5",
			b:           "2.4.4",
			expected:    1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			require.EqualValues(t, tc.expected, compareVersions(tc.a, tc.b))
		})
	}

}

func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		expected    string
	}{
		{
			description: "empty",
			input:       "",
			expected:    "v0.0.0",
		},
		{
			description: "major is 0",
			input:       "v0.1.2",
			expected:    "v0.1.2",
		},
		{
			description: "major only",
			input:       "1",
			expected:    "v1.0.0",
		},
		{
			description: "major and minor only",
			input:       "1.1",
			expected:    "v1.1.0",
		},
		{
			description: "zero-padded version",
			input:       "01.02.03",
			expected:    "v1.2.3",
		},
		{
			description: "valid semantic version",
			input:       "v1.2.3-4+567",
			expected:    "v1.2.3-4+567",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			output := normalizeVersion(tc.input)
			require.EqualValues(t, tc.expected, output)
		})
	}
}
