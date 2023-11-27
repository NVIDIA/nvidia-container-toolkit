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
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestWithWithDriverDotSoSymlinks(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
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
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
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
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
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
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
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
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
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
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{
						"nvidia-cdi-hook", "create-symlinks",
						"--link", "libcuda.so.1::/usr/lib/libcuda.so",
						"--link", "libGLX_nvidia.so.1.2.3::/usr/lib/libGLX_indirect.so.0",
						"--link", "libnvidia-opticalflow.so.1::/usr/lib/libnvidia-opticalflow.so",
					},
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
	}

	hookCreator := NewHookCreator()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := WithDriverDotSoSymlinks(
				logger,
				tc.discover,
				tc.version,
				hookCreator,
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

func TestGetDotSoSymlinks(t *testing.T) {
	testCases := []struct {
		description          string
		hostLibraryPath      string
		containerLibraryPath string
		getSonameFunc        func(string) (string, error)
		linkExistsFunc       func(string) (bool, error)
		expectedError        error
		expectedSymlinks     []string
	}{
		{
			description:     "libcuda.soname links",
			hostLibraryPath: "/usr/lib/libcuda.so.999.88.77",
			getSonameFunc: func(s string) (string, error) {
				return "libcuda.so.1", nil
			},
			expectedError: nil,
			expectedSymlinks: []string{
				"libcuda.so.999.88.77::/usr/lib/libcuda.so.1",
				"libcuda.so.1::/usr/lib/libcuda.so",
			},
		},
		{
			description:          "libcuda.soname links uses container path",
			hostLibraryPath:      "/usr/lib/libcuda.so.999.88.77",
			containerLibraryPath: "/some/container/path/libcuda.so.999.88.77",
			getSonameFunc: func(s string) (string, error) {
				return "libcuda.so.1", nil
			},
			expectedError: nil,
			expectedSymlinks: []string{
				"libcuda.so.999.88.77::/some/container/path/libcuda.so.1",
				"libcuda.so.1::/some/container/path/libcuda.so",
			},
		},
		{
			description:     "equal soname uses library path",
			hostLibraryPath: "/usr/lib/libcuda.so.999.88.77",
			getSonameFunc: func(s string) (string, error) {
				return "libcuda.so.999.88.77", nil
			},
			expectedError: nil,
			expectedSymlinks: []string{
				"libcuda.so.999.88.77::/usr/lib/libcuda.so",
			},
		},
		{
			description:     "nonexistent symlink is ignored",
			hostLibraryPath: "/usr/lib/libcuda.so.999.88.77",
			getSonameFunc: func(s string) (string, error) {
				return "libcuda.so.1", nil
			},
			expectedError: nil,
			linkExistsFunc: func(s string) (bool, error) {
				return strings.HasSuffix(s, "libcuda.so.1"), nil
			},
			expectedSymlinks: []string{
				"libcuda.so.999.88.77::/usr/lib/libcuda.so.1",
			},
		},
		{
			description:     "soname is skipped",
			hostLibraryPath: "/usr/lib/libcuda.so.999.88.77",
			getSonameFunc: func(s string) (string, error) {
				return "", nil
			},
			expectedError: nil,
			linkExistsFunc: func(s string) (bool, error) {
				return strings.HasSuffix(s, "libcuda.so"), nil
			},
			expectedSymlinks: []string{
				"libcuda.so.999.88.77::/usr/lib/libcuda.so",
			},
		},
	}

	for _, tc := range testCases {
		if tc.containerLibraryPath == "" {
			tc.containerLibraryPath = tc.hostLibraryPath
		}
		if tc.linkExistsFunc == nil {
			tc.linkExistsFunc = func(string) (bool, error) {
				return true, nil
			}
		}

		t.Run(tc.description, func(t *testing.T) {
			defer setGetSoname(tc.getSonameFunc)()
			defer setLinkExists(tc.linkExistsFunc)()

			sut := &additionalSymlinks{version: "*.*"}
			symlinks, err := sut.getDotSoSymlinks(tc.hostLibraryPath, tc.containerLibraryPath)

			if tc.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError.Error())
			}

			require.EqualValues(t, tc.expectedSymlinks, symlinks)
		})
	}
}

func TestGetSoLink(t *testing.T) {
	testCases := []struct {
		description    string
		input          string
		expectedSoLink string
	}{
		{
			description:    "empty string",
			input:          "",
			expectedSoLink: "",
		},
		{
			description:    "cuda driver library",
			input:          "libcuda.so.999.88.77",
			expectedSoLink: "libcuda.so",
		},
		{
			description:    "beta cuda driver library",
			input:          "libcuda.so.999.88",
			expectedSoLink: "libcuda.so",
		},
		{
			description:    "no .so in libname",
			input:          "foo.bar.baz",
			expectedSoLink: "",
		},
		{
			description:    "multiple .so in libname",
			input:          "foo.so.so.566",
			expectedSoLink: "foo.so.so",
		},
		{
			description:    "no suffix after so",
			input:          "foo.so",
			expectedSoLink: "foo.so",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			soLink := getSoLink(tc.input)

			require.Equal(t, tc.expectedSoLink, soLink)
		})
	}
}

func setGetSoname(override func(string) (string, error)) func() {
	original := getSoname
	getSoname = override

	return func() {
		getSoname = original
	}
}

func setLinkExists(override func(string) (bool, error)) func() {
	original := linkExists
	linkExists = override

	return func() {
		linkExists = original
	}
}
