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

package tegra_test

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestTegraNew(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	defer devices.SetAllForTest()()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testCases := []struct {
		description     string
		driverRootfs    string
		devRootfs       string
		expectedDevices []discover.Device
		expectedMounts  []discover.Mount
		expectedHooks   []discover.Hook
	}{
		{
			description:  "empty rootfs returns no devices",
			driverRootfs: "rootfs-empty",
		},
		{
			description:  "rootfs with device nodes returns devices",
			driverRootfs: "rootfs-orin",
			expectedDevices: []discover.Device{
				{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
			},
			expectedMounts: []discover.Mount{
				{Path: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", HostPath: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
				{Path: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", HostPath: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
		{
			description:  "split rootfs",
			driverRootfs: "rootfs-orin",
			devRootfs:    "rootfs-split/dev-root",
			expectedDevices: []discover.Device{
				{Path: "/dev/nvidia0", HostPath: "/dev/nvidia0"},
				{Path: "/dev/nvidiactl", HostPath: "/dev/nvidiactl"},
			},
			expectedMounts: []discover.Mount{
				{Path: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", HostPath: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
				{Path: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", HostPath: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
	}

	hookCreator := discover.NewHookCreator()

	// We use the same csv file list for all test cases.
	var csvFiles []string
	for _, csvFile := range csv.DefaultFileList() {
		csvFiles = append(csvFiles, filepath.Join(lookupRoot, "rootfs-orin", csvFile))
	}
	mountSpecs := tegra.MountSpecsFromCSVFiles(logger, csvFiles...)

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
			driver := root.New(root.WithDriverRoot(driverRoot), root.WithDevRoot(devRoot))
			d, err := tegra.New(
				tegra.WithLogger(logger),
				tegra.WithDriver(driver),
				tegra.WithHookCreator(hookCreator),
				tegra.WithMountSpecs(mountSpecs),
			)
			require.NoError(t, err)

			devices, err := d.Devices()
			require.NoError(t, err)

			if devRoot == "" {
				devRoot = driverRoot
			}
			require.EqualValues(t, tc.expectedDevices, test.StripRoot(devices, devRoot))

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMounts, test.StripRoot(mounts, driverRoot))

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedHooks, test.StripRoot(hooks, driverRoot))

			envVars, err := d.EnvVars()
			require.NoError(t, err)
			require.Empty(t, envVars)
		})
	}
}

func TestOptions(t *testing.T) {
	testCases := []struct {
		description         string
		driver              *root.Driver
		expectedErrorstring string
	}{
		{
			description:         "nill driver returns an error",
			expectedErrorstring: "a driver must be specified",
		},
		{
			description: "valid driver",
			driver:      root.New(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := tegra.New(
				tegra.WithDriver(tc.driver),
			)
			if tc.expectedErrorstring == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErrorstring)
			}
		})
	}
}
