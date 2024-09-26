/**
# Copyright 2024 NVIDIA CORPORATION
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

package discover

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithWithDriverDotSoSymlinks(t *testing.T) {
	testCases := []struct {
		description          string
		discover             Discover
		version              string
		expectedDevices      []Device
		expectedDevicesError error
		expectedHooks        []Hook
		expectedHooksError   error
		expectedMounts       []Mount
		expectedMountsError  error
	}{
		{
			description: "empty discoverer remains empty",
			discover:    None{},
		},
		{
			description: "non-matching discoverer remains unchanged",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					devices := []Device{
						{
							Path: "/dev/dev1",
						},
					}
					return devices, nil
				},
				HooksFunc: func() ([]Hook, error) {
					hooks := []Hook{
						{
							Lifecycle: "prestart",
							Path:      "/path/to/a/hook",
							Args:      []string{"hook", "arg1", "arg2"},
						},
					}
					return hooks, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libnotcuda.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			expectedDevices: []Device{
				{
					Path: "/dev/dev1",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "prestart",
					Path:      "/path/to/a/hook",
					Args:      []string{"hook", "arg1", "arg2"},
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libnotcuda.so.1.2.3",
				},
			},
		},
		{
			description: "libcuda.so.RM_VERSION is matched",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					return nil, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			version: "1.2.3",
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2.3",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/path/to/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
				},
			},
		},
		{
			description: "libcuda.so.RM_VERSION is matched by pattern",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					return nil, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			version: "",
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2.3",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/path/to/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
				},
			},
		},
		{
			description: "beta libcuda.so.RM_VERSION is matched",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					return nil, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/path/to/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
				},
			},
		},
		{
			description: "non-matching libcuda.so.RM_VERSION is ignored",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					return nil, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			version: "4.5.6",
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2.3",
				},
			},
		},
		{
			description: "hooks are extended",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					hooks := []Hook{
						{
							Lifecycle: "prestart",
							Path:      "/path/to/a/hook",
							Args:      []string{"hook", "arg1", "arg2"},
						},
					}
					return hooks, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			version: "1.2.3",
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2.3",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "prestart",
					Path:      "/path/to/a/hook",
					Args:      []string{"hook", "arg1", "arg2"},
				},
				{
					Lifecycle: "createContainer",
					Path:      "/path/to/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
				},
			},
		},
		{
			description: "all driver so symlinks are matched",
			discover: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					return nil, nil
				},
				HooksFunc: func() ([]Hook, error) {
					return nil, nil
				},
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib/libcuda.so.1.2.3",
						},
						{
							Path: "/usr/lib/libGLX_nvidia.so.1.2.3",
						},
						{
							Path: "/usr/lib/libnvidia-opticalflow.so.1.2.3",
						},
						{
							Path: "/usr/lib/libanother.so.1.2.3",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib/libcuda.so.1.2.3",
				},
				{
					Path: "/usr/lib/libGLX_nvidia.so.1.2.3",
				},
				{
					Path: "/usr/lib/libnvidia-opticalflow.so.1.2.3",
				},
				{
					Path: "/usr/lib/libanother.so.1.2.3",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/path/to/nvidia-cdi-hook",
					Args: []string{
						"nvidia-cdi-hook", "create-symlinks",
						"--link", "libcuda.so.1::/usr/lib/libcuda.so",
						"--link", "libGLX_nvidia.so.1.2.3::/usr/lib/libGLX_indirect.so.0",
						"--link", "libnvidia-opticalflow.so.1::/usr/lib/libnvidia-opticalflow.so",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := WithDriverDotSoSymlinks(
				tc.discover,
				tc.version,
				"/path/to/nvidia-cdi-hook",
			)

			devices, err := d.Devices()
			require.ErrorIs(t, err, tc.expectedDevicesError)
			require.EqualValues(t, tc.expectedDevices, devices)

			hooks, err := d.Hooks()
			require.ErrorIs(t, err, tc.expectedHooksError)
			require.EqualValues(t, tc.expectedHooks, hooks)

			mounts, err := d.Mounts()
			require.ErrorIs(t, err, tc.expectedMountsError)
			require.EqualValues(t, tc.expectedMounts, mounts)
		})
	}
}
