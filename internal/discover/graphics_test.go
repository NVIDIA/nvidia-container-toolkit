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

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestGraphicsLibrariesDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description    string
		libraries      *DiscoverMock
		expectedMounts []Mount
		expectedHooks  []Hook
	}{
		{
			description: "none discovered",
			libraries: &DiscoverMock{
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib64/libnvidia-egl-gbm.so.123.45.67",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib64/libnvidia-egl-gbm.so.123.45.67",
				},
			},
		},
		{
			description: "libnvidia-allocator discovered",
			libraries: &DiscoverMock{
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib64/libnvidia-allocator.so.123.45.67",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: nil,
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{"nvidia-cdi-hook", "create-symlinks",
						"--link", "../libnvidia-allocator.so.1::/usr/lib64/gbm/nvidia-drm_gbm.so",
					},
				},
			},
		},
		{
			description: "libnvidia-vulkan-producer discovered",
			libraries: &DiscoverMock{
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib64/libnvidia-vulkan-producer.so.123.45.67",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib64/libnvidia-vulkan-producer.so.123.45.67",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{"nvidia-cdi-hook", "create-symlinks",
						"--link", "libnvidia-vulkan-producer.so.123.45.67::/usr/lib64/libnvidia-vulkan-producer.so",
					},
				},
			},
		},
		{
			description: "libnvidia-allocator and libnvidia-vulkan-producer discovered",
			libraries: &DiscoverMock{
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib64/libnvidia-allocator.so.123.45.67",
						},
						{
							Path: "/usr/lib64/libnvidia-vulkan-producer.so.123.45.67",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib64/libnvidia-vulkan-producer.so.123.45.67",
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{"nvidia-cdi-hook", "create-symlinks",
						"--link", "../libnvidia-allocator.so.1::/usr/lib64/gbm/nvidia-drm_gbm.so",
						"--link", "libnvidia-vulkan-producer.so.123.45.67::/usr/lib64/libnvidia-vulkan-producer.so",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := &graphicsDriverLibraries{
				Discover:          tc.libraries,
				logger:            logger,
				nvidiaCDIHookPath: "/usr/bin/nvidia-cdi-hook",
			}

			devices, err := d.Devices()
			require.NoError(t, err)
			require.Empty(t, devices)
			require.Len(t, tc.libraries.calls.Devices, 1)

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMounts, mounts)
			require.Len(t, tc.libraries.calls.Mounts, 1)

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedHooks, hooks)
			require.Len(t, tc.libraries.calls.Mounts, 2)
			require.Len(t, tc.libraries.calls.Hooks, 0)
		})
	}
}
