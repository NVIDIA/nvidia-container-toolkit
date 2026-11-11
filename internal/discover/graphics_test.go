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
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestGraphicsLibrariesDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
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
			description: "libnvidia-allocator not filtered out when version does not equal driver version",
			libraries: &DiscoverMock{
				MountsFunc: func() ([]Mount, error) {
					mounts := []Mount{
						{
							Path: "/usr/lib64/libnvidia-allocator.so.999.99.99",
						},
					}
					return mounts, nil
				},
			},
			expectedMounts: []Mount{
				{
					Path: "/usr/lib64/libnvidia-allocator.so.999.99.99",
				},
			},
			expectedHooks: nil,
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
				Discover:      tc.libraries,
				logger:        logger,
				hookCreator:   hookCreator,
				driverVersion: "123.45.67",
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

// TestGraphicsLibrariesDiscovererInANestedDriverRoot covers the scenario in
// issue 554: a driver root that is not / and whose libraries are not under
// /usr/lib/x86_64-linux-gnu, as happens when the toolkit runs inside a snap and
// the host filesystem is mounted at /var/lib/snapd/hostfs. The X.Org search
// paths are derived from the directories the versioned driver libraries were
// actually found in, so the modules have to be discovered whatever the arch
// directory is called, and the driver root has to be prepended exactly once.
func TestGraphicsLibrariesDiscovererInANestedDriverRoot(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	hookCreator := NewHookCreator()

	const driverVersion = "999.88.77"

	testCases := []struct {
		description string
		libDir      string
	}{
		{
			description: "x86_64 arch directory",
			libDir:      "/usr/lib/x86_64-linux-gnu",
		},
		{
			description: "aarch64 arch directory",
			libDir:      "/usr/lib/aarch64-linux-gnu",
		},
		{
			description: "lib64",
			libDir:      "/usr/lib64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// A snap sees the host filesystem under a prefix, so the driver
			// root is a directory rather than /. EvalSymlinks because the
			// lookup resolves the paths it finds, and RelativeToRoot trims the
			// root as a string: on a system where the temporary directory sits
			// behind a symlink (macOS puts /var behind /private/var) an
			// unresolved root would not match and the root would be prepended
			// twice.
			tempDir, err := filepath.EvalSymlinks(t.TempDir())
			require.NoError(t, err)
			hostfs := filepath.Join(tempDir, "var", "lib", "snapd", "hostfs")

			files := []string{
				filepath.Join(tc.libDir, "libcuda.so."+driverVersion),
				filepath.Join(tc.libDir, "nvidia", "xorg", "nvidia_drv.so"),
				filepath.Join(tc.libDir, "nvidia", "xorg", "libglxserver_nvidia.so."+driverVersion),
			}
			for _, f := range files {
				path := filepath.Join(hostfs, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte{}, 0600))
			}

			driver := root.New(
				root.WithLogger(logger),
				root.WithDriverRoot(hostfs),
			)

			d, err := newGraphicsLibrariesDiscoverer(logger, driver, hookCreator)
			require.NoError(t, err)

			mounts, err := d.Mounts()
			require.NoError(t, err)

			discovered := make(map[string]string)
			for _, mount := range mounts {
				// A host path that still carries the driver root means the
				// prefix was added once, which is what the container needs.
				require.True(t, strings.HasPrefix(mount.HostPath, hostfs), "host path %q is outside the driver root", mount.HostPath)
				discovered[strings.TrimPrefix(mount.HostPath, hostfs)] = mount.Path
			}

			require.EqualValues(t, map[string]string{
				filepath.Join(tc.libDir, "nvidia", "xorg", "nvidia_drv.so"):                         filepath.Join(tc.libDir, "nvidia", "xorg", "nvidia_drv.so"),
				filepath.Join(tc.libDir, "nvidia", "xorg", "libglxserver_nvidia.so."+driverVersion): filepath.Join(tc.libDir, "nvidia", "xorg", "libglxserver_nvidia.so."+driverVersion),
			}, discovered)
		})
	}
}

func TestGraphicsConfigsDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description string
		files       []string
		// expected maps the path of the file in the driver root to the path
		// that it is expected to be mounted at in the container.
		expected map[string]string
	}{
		{
			description: "config files in the standard locations",
			files: []string{
				"/usr/share/glvnd/egl_vendor.d/10_nvidia.json",
				"/usr/share/egl/egl_external_platform.d/20_nvidia_xcb.json",
				"/usr/share/X11/xorg.conf.d/10-nvidia.conf",
				"/etc/OpenCL/vendors/nvidia.icd",
			},
			expected: map[string]string{
				"/usr/share/glvnd/egl_vendor.d/10_nvidia.json":              "/usr/share/glvnd/egl_vendor.d/10_nvidia.json",
				"/usr/share/egl/egl_external_platform.d/20_nvidia_xcb.json": "/usr/share/egl/egl_external_platform.d/20_nvidia_xcb.json",
				"/usr/share/X11/xorg.conf.d/10-nvidia.conf":                 "/usr/share/X11/xorg.conf.d/10-nvidia.conf",
				"/etc/OpenCL/vendors/nvidia.icd":                            "/etc/OpenCL/vendors/nvidia.icd",
			},
		},
		{
			description: "config files in non-standard locations are mounted at the standard locations",
			files: []string{
				"/usr/local/share/glvnd/egl_vendor.d/10_nvidia.json",
				"/usr/local/share/egl/egl_external_platform.d/20_nvidia_xlib.json",
				"/etc/X11/xorg.conf.d/nvidia-drm-outputclass.conf",
			},
			expected: map[string]string{
				"/usr/local/share/glvnd/egl_vendor.d/10_nvidia.json":               "/usr/share/glvnd/egl_vendor.d/10_nvidia.json",
				"/usr/local/share/egl/egl_external_platform.d/20_nvidia_xlib.json": "/usr/share/egl/egl_external_platform.d/20_nvidia_xlib.json",
				"/etc/X11/xorg.conf.d/nvidia-drm-outputclass.conf":                 "/usr/share/X11/xorg.conf.d/nvidia-drm-outputclass.conf",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// The config search paths include the XDG data dirs. These are
			// set explicitly to ensure that the test is not affected by the
			// environment that it is run in.
			t.Setenv("XDG_DATA_DIRS", "/usr/local/share:/usr/share")

			driverRoot := t.TempDir()
			for _, f := range tc.files {
				path := filepath.Join(driverRoot, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte{}, 0600))
			}

			driver := root.New(
				root.WithLogger(logger),
				root.WithDriverRoot(driverRoot),
			)

			mounts, err := newGraphicsConfigsDiscoverer(logger, driver).Mounts()
			require.NoError(t, err)

			discovered := make(map[string]string)
			for _, mount := range mounts {
				hostPath := strings.TrimPrefix(mount.HostPath, driverRoot)
				discovered[hostPath] = mount.Path
			}
			require.EqualValues(t, tc.expected, discovered)
		})
	}
}

func TestDrmDevicesByPath(t *testing.T) {
	defer devices.SetAllForTest()()
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)
	devRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-drm")

	logger, _ := testlog.NewNullLogger()
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

func TestIsDriverLibrary(t *testing.T) {
	testCases := []struct {
		description    string
		filename       string
		libraryName    string
		driverVersion  string
		expectedResult bool
	}{
		{
			description:    "driver library file matched",
			libraryName:    "libnvidia-vulkan-producer.so",
			filename:       "libnvidia-vulkan-producer.so.123.45.67",
			driverVersion:  "123.45.67",
			expectedResult: true,
		},
		{
			description:    "driver library file matched with extraneous \".\"",
			libraryName:    "libnvidia-vulkan-producer.so.",
			filename:       "libnvidia-vulkan-producer.so.123.45.67",
			driverVersion:  "123.45.67",
			expectedResult: true,
		},
		{
			description:    "driver library file not matched due to mismatching driver version",
			libraryName:    "libnvidia-vulkan-producer.so",
			filename:       "libnvidia-vulkan-producer.so.123.45.67",
			driverVersion:  "999.99.99",
			expectedResult: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			gdl := graphicsDriverLibraries{
				driverVersion: tc.driverVersion,
			}
			result := gdl.isDriverLibrary(tc.filename, tc.libraryName)
			require.Equal(t, tc.expectedResult, result)
		})

	}
}
