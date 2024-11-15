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

package image

import (
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestParseMajorMinorVersionValid(t *testing.T) {
	var tests = []struct {
		version  string
		expected string
	}{
		{"0", "0.0"},
		{"8", "8.0"},
		{"7.5", "7.5"},
		{"9.0.116", "9.0"},
		{"4294967295.4294967295.4294967295", "4294967295.4294967295"},
		{"v11.6", "11.6"},
	}
	for _, c := range tests {
		t.Run(c.version, func(t *testing.T) {
			version, err := parseMajorMinorVersion(c.version)

			require.NoError(t, err)
			require.Equal(t, c.expected, version)
		})
	}
}

func TestParseMajorMinorVersionInvalid(t *testing.T) {
	var tests = []string{
		"foo",
		"foo.5.10",
		"9.0.116.50",
		"9.0.116foo",
		"7.foo",
		"9.0.bar",
		"9.4294967296",
		"9.0.116.",
		"9..0",
		"9.",
		".5.10",
		"-9",
		"+9",
		"-9.1.116",
		"-9.-1.-116",
	}
	for _, c := range tests {
		t.Run(c, func(t *testing.T) {
			_, err := parseMajorMinorVersion(c)
			require.Error(t, err)
		})
	}
}

func TestGetRequirements(t *testing.T) {
	testCases := []struct {
		description  string
		env          []string
		requirements []string
	}{
		{
			description:  "NVIDIA_REQUIRE_JETPACK is ignored",
			env:          []string{"NVIDIA_REQUIRE_JETPACK=csv-mounts=all"},
			requirements: nil,
		},
		{
			description:  "NVIDIA_REQUIRE_JETPACK_HOST_MOUNTS is ignored",
			env:          []string{"NVIDIA_REQUIRE_JETPACK_HOST_MOUNTS=base-only"},
			requirements: nil,
		},
		{
			description:  "single requirement set",
			env:          []string{"NVIDIA_REQUIRE_CUDA=cuda>=11.6"},
			requirements: []string{"cuda>=11.6"},
		},
		{
			description:  "requirements are concatenated requirement set",
			env:          []string{"NVIDIA_REQUIRE_CUDA=cuda>=11.6", "NVIDIA_REQUIRE_BRAND=brand=tesla"},
			requirements: []string{"cuda>=11.6", "brand=tesla"},
		},
		{
			description:  "legacy image",
			env:          []string{"CUDA_VERSION=11.6"},
			requirements: []string{"cuda>=11.6"},
		},
		{
			description:  "legacy image with additional requirement",
			env:          []string{"CUDA_VERSION=11.6", "NVIDIA_REQUIRE_BRAND=brand=tesla"},
			requirements: []string{"cuda>=11.6", "brand=tesla"},
		},
		{
			description:  "NVIDIA_DISABLE_REQUIRE ignores requirements",
			env:          []string{"NVIDIA_REQUIRE_CUDA=cuda>=11.6", "NVIDIA_REQUIRE_BRAND=brand=tesla", "NVIDIA_DISABLE_REQUIRE=true"},
			requirements: []string{},
		},
		{
			description:  "NVIDIA_DISABLE_REQUIRE ignores legacy image requirements",
			env:          []string{"CUDA_VERSION=11.6", "NVIDIA_REQUIRE_BRAND=brand=tesla", "NVIDIA_DISABLE_REQUIRE=true"},
			requirements: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			image, err := NewCUDAImageFromEnv(tc.env)
			require.NoError(t, err)

			requirements, err := image.GetRequirements()
			require.NoError(t, err)
			require.ElementsMatch(t, tc.requirements, requirements)
		})

	}
}

func TestGetVisibleDevicesFromMounts(t *testing.T) {
	var tests = []struct {
		description     string
		mounts          []specs.Mount
		expectedDevices []string
	}{
		{
			description:     "No mounts",
			mounts:          nil,
			expectedDevices: nil,
		},
		{
			description: "Host path is not /dev/null",
			mounts: []specs.Mount{
				{
					Source:      "/not/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU0"),
				},
			},
			expectedDevices: nil,
		},
		{
			description: "Container path is not prefixed by 'root'",
			mounts: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join("/other/prefix", "GPU0"),
				},
			},
			expectedDevices: nil,
		},
		{
			description: "Container path is only 'root'",
			mounts: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: DeviceListAsVolumeMountsRoot,
				},
			},
			expectedDevices: nil,
		},
		{
			description:     "Discover 2 devices",
			mounts:          makeTestMounts("GPU0", "GPU1"),
			expectedDevices: []string{"GPU0", "GPU1"},
		},
		{
			description:     "Discover 2 devices with slashes in the name",
			mounts:          makeTestMounts("GPU0-MIG0/0/1", "GPU1-MIG0/0/1"),
			expectedDevices: []string{"GPU0-MIG0/0/1", "GPU1-MIG0/0/1"},
		},
		{
			description:     "cdi devices are ignored",
			mounts:          makeTestMounts("GPU0", "cdi/nvidia.com/gpu=all", "GPU1"),
			expectedDevices: []string{"GPU0", "GPU1"},
		},
		{
			description:     "imex devices are ignored",
			mounts:          makeTestMounts("GPU0", "imex/0", "GPU1"),
			expectedDevices: []string{"GPU0", "GPU1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := New(WithMounts(tc.mounts))
			require.Equal(t, tc.expectedDevices, image.VisibleDevicesFromMounts())
		})
	}
}

func TestImexChannelsFromEnvVar(t *testing.T) {
	testCases := []struct {
		description string
		env         []string
		expected    []string
	}{
		{
			description: "no imex channels specified",
		},
		{
			description: "imex channel specified",
			env: []string{
				"NVIDIA_IMEX_CHANNELS=3,4",
			},
			expected: []string{"3", "4"},
		},
	}

	for _, tc := range testCases {
		for id, baseEnvvars := range map[string][]string{"": nil, "legacy": {"CUDA_VERSION=1.2.3"}} {
			t.Run(tc.description+id, func(t *testing.T) {
				i, err := NewCUDAImageFromEnv(append(baseEnvvars, tc.env...))
				require.NoError(t, err)

				channels := i.ImexChannelsFromEnvVar()
				require.EqualValues(t, tc.expected, channels)
			})
		}
	}
}

func makeTestMounts(paths ...string) []specs.Mount {
	var mounts []specs.Mount
	for _, path := range paths {
		mount := specs.Mount{
			Source:      "/dev/null",
			Destination: filepath.Join(DeviceListAsVolumeMountsRoot, path),
		}
		mounts = append(mounts, mount)
	}
	return mounts
}
