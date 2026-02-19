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

package discover_test

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestNewGDSDiscoverer(t *testing.T) {
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
	}{
		{
			description:  "empty rootfs returns no devices",
			driverRootfs: "rootfs-empty",
		},
		{
			description:  "single rootfs",
			driverRootfs: "rootfs-1",
			expectedDevices: []discover.Device{
				{Path: "/dev/nvidia-fs0", HostPath: "/dev/nvidia-fs0"},
			},
			expectedMounts: []discover.Mount{
				{Path: "/run/udev", HostPath: "/run/udev", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
				{Path: "/etc/cufile.json", HostPath: "/etc/cufile.json", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
			},
		},
		{
			description:  "split rootfs",
			driverRootfs: "rootfs-split/driver-root",
			devRootfs:    "rootfs-split/dev-root",
			expectedDevices: []discover.Device{
				{Path: "/dev/nvidia-fs0", HostPath: "/dev/nvidia-fs0"},
			},
			expectedMounts: []discover.Mount{
				{Path: "/run/udev", HostPath: "/run/udev", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
				{Path: "/etc/cufile.json", HostPath: "/etc/cufile.json", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
			},
		},
	}

	for _, tc := range testCases {
		if tc.devRootfs == "" {
			tc.devRootfs = tc.driverRootfs
		}
		t.Run(tc.description, func(t *testing.T) {
			driverRoot := filepath.Join(lookupRoot, tc.driverRootfs)
			devRoot := filepath.Join(lookupRoot, tc.devRootfs)
			driver := root.New(root.WithDriverRoot(driverRoot), root.WithDevRoot(devRoot))
			d, err := discover.NewGDSDiscoverer(logger, driver)
			require.NoError(t, err)

			devices, err := d.Devices()
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedDevices, test.StripRoot(devices, devRoot))

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMounts, test.StripRoot(mounts, driverRoot))

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.Empty(t, hooks)

			envVars, err := d.EnvVars()
			require.NoError(t, err)
			require.Empty(t, envVars)
		})
	}
}
