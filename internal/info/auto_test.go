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

package info

import (
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/info"
)

func TestResolveAutoMode(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description  string
		mode         string
		expectedMode string
		info         info.Interface
		image        image.CUDA
	}{
		{
			description:  "non-auto resolves to input",
			mode:         "not-auto",
			expectedMode: "not-auto",
		},
		{
			description: "nvml non-tegra resolves to legacy",
			mode:        "auto",
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return true, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return false, "tegra"
				},
			},
			expectedMode: "legacy",
		},
		{
			description: "non-nvml non-tegra resolves to legacy",
			mode:        "auto",
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return false, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return false, "tegra"
				},
			},
			expectedMode: "legacy",
		},
		{
			description: "nvml tegra resolves to legacy",
			mode:        "auto",
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return true, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return true, "tegra"
				},
			},
			expectedMode: "legacy",
		},
		{
			description: "non-nvml tegra resolves to csv",
			mode:        "auto",
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return false, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return true, "tegra"
				},
			},
			expectedMode: "csv",
		},
		{
			description:  "cdi devices resolves to cdi",
			mode:         "auto",
			expectedMode: "cdi",
			image: image.CUDA{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=all",
			},
		},
		{
			description:  "multiple cdi devices resolves to cdi",
			mode:         "auto",
			expectedMode: "cdi",
			image: image.CUDA{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,nvidia.com/gpu=1",
			},
		},
		{
			description: "at least one non-cdi device resolves to legacy",
			mode:        "auto",
			image: image.CUDA{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,0",
			},
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return true, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return true, "tegra"
				},
			},
			expectedMode: "legacy",
		},
		{
			description: "at least one non-cdi device resolves to csv",
			mode:        "auto",
			image: image.CUDA{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,0",
			},
			info: &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return false, "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return true, "tegra"
				},
			},
			expectedMode: "csv",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			r := resolver{
				logger: logger,
				info:   tc.info,
			}
			mode := r.resolveMode(tc.mode, tc.image)
			require.EqualValues(t, tc.expectedMode, mode)
		})
	}
}
