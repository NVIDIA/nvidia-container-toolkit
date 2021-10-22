/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	testCases := []struct {
		options                 options
		expectedDefaultRuntime  string
		expectedRuntimeBinaries map[string]string
	}{
		{
			expectedRuntimeBinaries: map[string]string{
				"nvidia":              "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: true,
			},
			expectedDefaultRuntime: "nvidia",
			expectedRuntimeBinaries: map[string]string{
				"nvidia":              "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: true,
				runtimeClass: "nvidia",
			},
			expectedDefaultRuntime: "nvidia",
			expectedRuntimeBinaries: map[string]string{
				"nvidia":              "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: true,
				runtimeClass: "NAME",
			},
			expectedDefaultRuntime: "NAME",
			expectedRuntimeBinaries: map[string]string{
				"NAME":                "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: false,
				runtimeClass: "NAME",
			},
			expectedRuntimeBinaries: map[string]string{
				"NAME":                "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: true,
				runtimeClass: "nvidia-experimental",
			},
			expectedDefaultRuntime: "nvidia-experimental",
			expectedRuntimeBinaries: map[string]string{
				"nvidia":              "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
		{
			options: options{
				setAsDefault: false,
				runtimeClass: "nvidia-experimental",
			},
			expectedRuntimeBinaries: map[string]string{
				"nvidia":              "nvidia-container-runtime",
				"nvidia-experimental": "nvidia-container-runtime-experimental",
			},
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.expectedDefaultRuntime, tc.options.getDefaultRuntime(), "%d: %v", i, tc)
		require.EqualValues(t, tc.expectedRuntimeBinaries, tc.options.getRuntimeBinaries(), "%d: %v", i, tc)
	}
}
