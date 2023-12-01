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

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestResolveAutoMode(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description  string
		mode         string
		expectedMode string
		info         map[string]bool
		envmap       map[string]string
		mounts       []string
	}{
		{
			description:  "non-auto resolves to input",
			mode:         "not-auto",
			expectedMode: "not-auto",
		},
		{
			description:  "no info defaults to legacy",
			mode:         "auto",
			info:         map[string]bool{},
			expectedMode: "legacy",
		},
		{
			description: "non-nvml, non-tegra, nvgpu resolves to csv",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  false,
				"tegra": false,
				"nvgpu": true,
			},
			expectedMode: "csv",
		},
		{
			description: "non-nvml, tegra, non-nvgpu resolves to csv",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  false,
				"tegra": true,
				"nvgpu": false,
			},
			expectedMode: "csv",
		},
		{
			description: "non-nvml, tegra, nvgpu resolves to csv",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  false,
				"tegra": true,
				"nvgpu": true,
			},
			expectedMode: "csv",
		},
		{
			description: "nvml, non-tegra, non-nvgpu resolves to legacy",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  true,
				"tegra": false,
				"nvgpu": false,
			},
			expectedMode: "legacy",
		},
		{
			description: "nvml, non-tegra, nvgpu resolves to csv",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  true,
				"tegra": false,
				"nvgpu": true,
			},
			expectedMode: "csv",
		},
		{
			description: "nvml, tegra, non-nvgpu resolves to legacy",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  true,
				"tegra": true,
				"nvgpu": false,
			},
			expectedMode: "legacy",
		},
		{
			description: "nvml, tegra, nvgpu resolves to csv",
			mode:        "auto",
			info: map[string]bool{
				"nvml":  true,
				"tegra": true,
				"nvgpu": true,
			},
			expectedMode: "csv",
		},
		{
			description:  "cdi devices resolves to cdi",
			mode:         "auto",
			expectedMode: "cdi",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=all",
			},
		},
		{
			description:  "multiple cdi devices resolves to cdi",
			mode:         "auto",
			expectedMode: "cdi",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,nvidia.com/gpu=1",
			},
		},
		{
			description: "at least one non-cdi device resolves to legacy",
			mode:        "auto",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,0",
			},
			info: map[string]bool{
				"nvml":  true,
				"tegra": false,
				"nvgpu": false,
			},
			expectedMode: "legacy",
		},
		{
			description: "at least one non-cdi device resolves to csv",
			mode:        "auto",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "nvidia.com/gpu=0,0",
			},
			info: map[string]bool{
				"nvml":  false,
				"tegra": true,
				"nvgpu": false,
			},
			expectedMode: "csv",
		},
		{
			description: "cdi mount devices resolves to CDI",
			mode:        "auto",
			mounts: []string{
				"/var/run/nvidia-container-devices/cdi/nvidia.com/gpu/0",
			},
			expectedMode: "cdi",
		},
		{
			description: "cdi mount and non-CDI devices resolves to legacy",
			mode:        "auto",
			mounts: []string{
				"/var/run/nvidia-container-devices/cdi/nvidia.com/gpu/0",
				"/var/run/nvidia-container-devices/all",
			},
			info: map[string]bool{
				"nvml":  true,
				"tegra": false,
				"nvgpu": false,
			},
			expectedMode: "legacy",
		},
		{
			description: "cdi mount and non-CDI envvar resolves to legacy",
			mode:        "auto",
			envmap: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "0",
			},
			mounts: []string{
				"/var/run/nvidia-container-devices/cdi/nvidia.com/gpu/0",
			},
			info: map[string]bool{
				"nvml":  true,
				"tegra": false,
				"nvgpu": false,
			},
			expectedMode: "legacy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			info := &infoInterfaceMock{
				HasNvmlFunc: func() (bool, string) {
					return tc.info["nvml"], "nvml"
				},
				IsTegraSystemFunc: func() (bool, string) {
					return tc.info["tegra"], "tegra"
				},
				UsesNVGPUModuleFunc: func() (bool, string) {
					return tc.info["nvgpu"], "nvgpu"
				},
			}

			r := resolver{
				logger: logger,
				info:   info,
			}

			var mounts []specs.Mount
			for _, d := range tc.mounts {
				mount := specs.Mount{
					Source:      "/dev/null",
					Destination: d,
				}
				mounts = append(mounts, mount)
			}
			image, _ := image.New(
				image.WithEnvMap(tc.envmap),
				image.WithMounts(mounts),
			)
			mode := r.resolveMode(tc.mode, image)
			require.EqualValues(t, tc.expectedMode, mode)
		})
	}
}
