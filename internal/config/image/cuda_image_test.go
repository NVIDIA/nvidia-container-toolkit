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
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestNewCUDAImageFromSpec(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description string
		spec        *specs.Spec
		options     []Option
		expected    CUDA
	}{
		{
			description: "no env vars",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{},
				},
			},
			expected: CUDA{
				logger:                   logger,
				env:                      map[string]string{},
				acceptEnvvarUnprivileged: true,
			},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES=all",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=all"},
				},
			},
			expected: CUDA{
				logger:                   logger,
				env:                      map[string]string{"NVIDIA_VISIBLE_DEVICES": "all"},
				acceptEnvvarUnprivileged: true,
			},
		},
		{
			description: "Spec overrides options",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=all"},
				},
				Mounts: []specs.Mount{
					{
						Source:      "/spec-source",
						Destination: "/spec-destination",
					},
				},
			},
			options: []Option{
				WithEnvMap(map[string]string{"OTHER": "value"}),
				WithMounts([]specs.Mount{
					{
						Source:      "/option-source",
						Destination: "/option-destination",
					},
				}),
			},
			expected: CUDA{
				logger: logger,
				env:    map[string]string{"NVIDIA_VISIBLE_DEVICES": "all"},
				mounts: []specs.Mount{
					{
						Source:      "/spec-source",
						Destination: "/spec-destination",
					},
				},
				acceptEnvvarUnprivileged: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			options := append([]Option{WithLogger(logger)}, tc.options...)
			image, err := NewCUDAImageFromSpec(tc.spec, options...)
			require.NoError(t, err)
			require.EqualValues(t, tc.expected, image)
		})
	}
}

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
			image, err := newCUDAImageFromEnv(tc.env)
			require.NoError(t, err)

			requirements, err := image.GetRequirements()
			require.NoError(t, err)
			require.ElementsMatch(t, tc.requirements, requirements)
		})

	}
}

func TestGetDevicesFromEnvvar(t *testing.T) {
	envDockerResourceGPUs := "DOCKER_RESOURCE_GPUS"
	gpuID := "GPU-12345"
	anotherGPUID := "GPU-67890"
	thirdGPUID := "MIG-12345"

	var tests = []struct {
		description                   string
		preferredVisibleDeviceEnvVars []string
		env                           map[string]string
		expectedDevices               []string
	}{
		{
			description: "empty env returns nil for non-legacy image",
		},
		{
			description: "blank NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "",
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "void",
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "none",
			},
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: gpuID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: gpuID,
				EnvVarCudaVersion:          "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				EnvVarCudaVersion: "legacy",
			},
			expectedDevices: []string{"all"},
		},
		// Add the `DOCKER_RESOURCE_GPUS` envvar and ensure that this is ignored when
		// not enabled
		{
			description: "missing NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
		},
		{
			description: "blank NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "",
				envDockerResourceGPUs:      anotherGPUID,
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "void",
				envDockerResourceGPUs:      anotherGPUID,
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: "none",
				envDockerResourceGPUs:      anotherGPUID,
			},
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:      anotherGPUID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:      anotherGPUID,
				EnvVarCudaVersion:          "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
				EnvVarCudaVersion:     "legacy",
			},
			expectedDevices: []string{"all"},
		},
		// Add the `DOCKER_RESOURCE_GPUS` envvar and ensure that this is selected when
		// enabled
		{
			description:                   "empty env returns nil for non-legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
		},
		{
			description:                   "blank DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "",
			},
		},
		{
			description:                   "'void' DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "void",
			},
		},
		{
			description:                   "'none' DOCKER_RESOURCE_GPUS returns empty for non-legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "none",
			},
			expectedDevices: []string{""},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS set returns value for non-legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS set returns value for legacy image",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
				EnvVarCudaVersion:     "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS is selected if present",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS overrides NVIDIA_VISIBLE_DEVICES if present",
			preferredVisibleDeviceEnvVars: []string{envDockerResourceGPUs},
			env: map[string]string{
				EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:      anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS_ADDITIONAL overrides NVIDIA_VISIBLE_DEVICES if present",
			preferredVisibleDeviceEnvVars: []string{"DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				EnvVarNvidiaVisibleDevices:        gpuID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:                   "All available swarm resource envvars are selected and override NVIDIA_VISIBLE_DEVICES if present",
			preferredVisibleDeviceEnvVars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				EnvVarNvidiaVisibleDevices:        gpuID,
				"DOCKER_RESOURCE_GPUS":            thirdGPUID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{thirdGPUID, anotherGPUID},
		},
		{
			description:                   "DOCKER_RESOURCE_GPUS_ADDITIONAL or DOCKER_RESOURCE_GPUS override NVIDIA_VISIBLE_DEVICES if present",
			preferredVisibleDeviceEnvVars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				EnvVarNvidiaVisibleDevices:        gpuID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			image, err := New(
				WithEnvMap(tc.env),
				WithPrivileged(true),
				WithAcceptDeviceListAsVolumeMounts(false),
				WithAcceptEnvvarUnprivileged(false),
				WithPreferredVisibleDevicesEnvVars(tc.preferredVisibleDeviceEnvVars...),
			)

			require.NoError(t, err)
			devices := image.VisibleDevicesFromEnvVar()
			require.EqualValues(t, tc.expectedDevices, devices)
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
			image, err := New(WithMounts(tc.mounts))
			require.NoError(t, err)
			require.Equal(t, tc.expectedDevices, image.visibleDevicesFromMounts())
		})
	}
}

func TestVisibleDevices(t *testing.T) {
	var tests = []struct {
		description        string
		mountDevices       []specs.Mount
		envvarDevices      string
		privileged         bool
		acceptUnprivileged bool
		acceptMounts       bool
		expectedDevices    []string
	}{
		{
			description: "Mount devices, unprivileged, no accept unprivileged",
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    []string{"GPU0", "GPU1"},
		},
		{
			description:        "No mount devices, unprivileged, no accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    nil,
		},
		{
			description:        "No mount devices, privileged, no accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         true,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    []string{"GPU0", "GPU1"},
		},
		{
			description:        "No mount devices, unprivileged, accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         false,
			acceptUnprivileged: true,
			acceptMounts:       true,
			expectedDevices:    []string{"GPU0", "GPU1"},
		},
		{
			description: "Mount devices, unprivileged, accept unprivileged, no accept mounts",
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: true,
			acceptMounts:       false,
			expectedDevices:    []string{"GPU2", "GPU3"},
		},
		{
			description: "Mount devices, unprivileged, no accept unprivileged, no accept mounts",
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(DeviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       false,
			expectedDevices:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			// Wrap the call to getDevices() in a closure.
			image, err := New(
				WithEnvMap(
					map[string]string{
						EnvVarNvidiaVisibleDevices: tc.envvarDevices,
					},
				),
				WithMounts(tc.mountDevices),
				WithPrivileged(tc.privileged),
				WithAcceptDeviceListAsVolumeMounts(tc.acceptMounts),
				WithAcceptEnvvarUnprivileged(tc.acceptUnprivileged),
			)
			require.NoError(t, err)
			require.Equal(t, tc.expectedDevices, image.VisibleDevices())
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
				i, err := newCUDAImageFromEnv(append(baseEnvvars, tc.env...))
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
