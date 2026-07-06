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
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGetCUDACompatElfHeaderFromReader(t *testing.T) {
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	dataRoot := filepath.Join(moduleRoot, "testdata", "compat")

	testCases := []struct {
		description   string
		filename      string
		expected      *compatElfHeader
		expectedError string
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
		{
			description:   "invalid json",
			filename:      "libcuda.invalid.json.so.99.88",
			expectedError: "could not unmarshal JSON data",
		},
		{
			description: "orin-13.2.1",
			filename:    "libcuda.orin.13.2.1.so.1.1",
			expected: &compatElfHeader{
				Format:      1,
				CUDAVersion: "13.2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			libpath := filepath.Join(dataRoot, tc.filename)
			lib, err := os.Open(libpath)
			require.NoError(t, err)

			h, err := GetCUDACompatElfHeaderFromReader(lib)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				require.Nil(t, h)
				return
			}
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

	parse := func(v string) *semver.Version {
		if v == "" {
			return nil
		}
		sv, err := semver.NewVersion(v)
		require.NoError(t, err)
		return sv
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			useCompat := tc.elfHeader.UseCompat(parse(tc.compatDriverVersion), parse(tc.hostDriverVersion), parse(tc.hostCudaVersion))

			require.EqualValues(t, tc.expected, useCompat)
		})
	}
}
