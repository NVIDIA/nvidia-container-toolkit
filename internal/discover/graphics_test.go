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
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGraphicsLibrariesDiscoverer(t *testing.T) {
	logger := logger.Interface{Logger: testr.New(t)}
	hookCreator := NewHookCreator()

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
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
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
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
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
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := &graphicsDriverLibraries{
				Discover:    tc.libraries,
				logger:      logger,
				hookCreator: hookCreator,
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

func TestDrmDevicesByPath(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)
	devRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-drm")

	logger := logger.Interface{Logger: testr.New(t)}
	hookCreator := NewHookCreator()

	testCases := []struct {
		description   string
		devices       Discover
		devRoot       string
		expectedError error
		expectedHooks []Hook
	}{
		{
			description: "no devices",
			devices:     &DiscoverMock{},
		},
		{
			description: "single device",
			devices: &DiscoverMock{
				DevicesFunc: func() ([]Device, error) {
					devices := []Device{
						{
							HostPath: "/dev/dri/card0",
						},
						{
							HostPath: "/dev/dri/renderD128",
						},
					}
					return devices, nil
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{
						"nvidia-cdi-hook", "create-symlinks",
						"--link", "../card0::{{ .DevRoot }}/dev/dri/by-path/pci-0000:07:00.0-card",
						"--link", "../renderD128::{{ .DevRoot }}/dev/dri/by-path/pci-0000:07:00.0-render",
					},
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
	}

	for _, tc := range testCases {

		for _, h := range tc.expectedHooks {
			for i := range h.Args {
				h.Args[i] = strings.ReplaceAll(h.Args[i], "{{ .DevRoot }}", devRoot)
			}
		}

		t.Run(tc.description, func(t *testing.T) {
			d := newCreateDRMByPathSymlinks(logger, tc.devices, devRoot, hookCreator)

			devices, err := d.Devices()
			require.NoError(t, err)
			require.Empty(t, devices)

			envVars, err := d.EnvVars()
			require.NoError(t, err)
			require.Empty(t, envVars)

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.Empty(t, mounts)

			hooks, err := d.Hooks()
			require.EqualValues(t, tc.expectedError, err)

			require.EqualValues(t, tc.expectedHooks, hooks)
		})
	}
}
