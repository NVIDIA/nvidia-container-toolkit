/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package nvcdi

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestNew(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	defer devices.SetAllForTest()()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testCases := []struct {
		description       string
		mode              string
		driverRootfs      string
		devRootfs         string
		expectedInitError error

		expectedSpec      *specs.Spec
		expectedSpecError error
	}{
		{
			description:  "nvswitch mode is supported",
			mode:         "nvswitch",
			driverRootfs: "rootfs-with-nvswitch",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/nvswitch",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path:     "/dev/nvidia-nvswitch0",
									HostPath: "/dev/nvidia-nvswitch0",
								},
								{
									Path:     "/dev/nvidia-nvswitch1",
									HostPath: "/dev/nvidia-nvswitch1",
								},
								{
									Path:     "/dev/nvidia-nvswitchctl",
									HostPath: "/dev/nvidia-nvswitchctl",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=void",
					},
				},
			},
		},
		{
			description:  "gds mode is supported",
			mode:         "gds",
			driverRootfs: "rootfs-1",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/gds",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{Path: "/dev/nvidia-fs0", HostPath: "/dev/nvidia-fs0"},
							},
							Mounts: []*specs.Mount{
								{ContainerPath: "/etc/cufile.json", HostPath: "/etc/cufile.json", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
								{ContainerPath: "/run/udev", HostPath: "/run/udev", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=void",
					},
				},
			},
		},
		{
			description:  "gdrcopy mode is supported",
			mode:         "gdrcopy",
			driverRootfs: "rootfs-1",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/gdrcopy",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path:     "/dev/gdrdrv",
									HostPath: "/dev/gdrdrv",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=void",
					},
				},
			},
		},
		{
			description:  "mofed mode is supported",
			mode:         "mofed",
			driverRootfs: "rootfs-1",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/mofed",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{Path: "/dev/infiniband/rdma_cm", HostPath: "/dev/infiniband/rdma_cm"},
								{Path: "/dev/infiniband/uverbs0", HostPath: "/dev/infiniband/uverbs0"},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=void",
					},
				},
			},
		},
		{
			description:  "management mode is supported",
			mode:         "management",
			driverRootfs: "rootfs-1",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "management.nvidia.com/gpu",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
								{Path: "/dev/nvidiactl", HostPath: "/dev/nvidiactl"},
								{Path: "/dev/nvidia-caps-imex-channels/channel0", HostPath: "/dev/nvidia-caps-imex-channels/channel0"},
								{Path: "/dev/nvidia-caps-imex-channels/channel1", HostPath: "/dev/nvidia-caps-imex-channels/channel1"},
								{Path: "/dev/nvidia-caps-imex-channels/channel2047", HostPath: "/dev/nvidia-caps-imex-channels/channel2047"},
								{Path: "/dev/nvidia-caps/nvidia-cap1", HostPath: "/dev/nvidia-caps/nvidia-cap1"},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu",
						"NVIDIA_VISIBLE_DEVICES=void",
					},
					Mounts: []*specs.Mount{
						{ContainerPath: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77", HostPath: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
						{ContainerPath: "/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77", HostPath: "/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
					},
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/lib/x86_64-linux-gnu/libcuda.so"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/lib/x86_64-linux-gnu", "--folder", "/lib/x86_64-linux-gnu/vdpau"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
					},
				},
			},
		},
		{
			description:  "management mode is supported in split root",
			mode:         "management",
			driverRootfs: "rootfs-split/driver-root",
			devRootfs:    "rootfs-split/dev-root",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "management.nvidia.com/gpu",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
								{Path: "/dev/nvidiactl", HostPath: "/dev/nvidiactl"},
								{Path: "/dev/nvidia-caps-imex-channels/channel0", HostPath: "/dev/nvidia-caps-imex-channels/channel0"},
								{Path: "/dev/nvidia-caps-imex-channels/channel1", HostPath: "/dev/nvidia-caps-imex-channels/channel1"},
								{Path: "/dev/nvidia-caps-imex-channels/channel2047", HostPath: "/dev/nvidia-caps-imex-channels/channel2047"},
								{Path: "/dev/nvidia-caps/nvidia-cap1", HostPath: "/dev/nvidia-caps/nvidia-cap1"},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_CTK_LIBCUDA_DIR=/lib/x86_64-linux-gnu",
						"NVIDIA_VISIBLE_DEVICES=void",
					},
					Mounts: []*specs.Mount{
						{ContainerPath: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77", HostPath: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
						{ContainerPath: "/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77", HostPath: "/lib/x86_64-linux-gnu/vdpau/libvdpau_nvidia.so.999.88.77", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
					},
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/lib/x86_64-linux-gnu/libcuda.so"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
						{
							HookName: "createContainer",
							Path:     "/usr/bin/nvidia-cdi-hook",
							Args:     []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/lib/x86_64-linux-gnu", "--folder", "/lib/x86_64-linux-gnu/vdpau"},
							Env:      []string{"NVIDIA_CTK_DEBUG=false"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			driverRoot := tc.driverRootfs
			if driverRoot != "" {
				driverRoot = filepath.Join(lookupRoot, tc.driverRootfs)
			}
			devRoot := tc.devRootfs
			if devRoot != "" {
				devRoot = filepath.Join(lookupRoot, tc.devRootfs)
			}

			l, err := New(
				WithLogger(logger),
				WithMode(tc.mode),
				WithDriverRoot(driverRoot),
				WithDevRoot(devRoot),
			)
			require.EqualValues(t, tc.expectedInitError, err)

			s, err := l.GetSpec()
			require.EqualValues(t, tc.expectedSpecError, err)

			require.EqualValues(t, tc.expectedSpec, test.StripRoot(test.StripRoot(s.Raw(), driverRoot), devRoot))
		})
	}
}
