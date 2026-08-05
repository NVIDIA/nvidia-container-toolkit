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

package discover

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestNewMigCapDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	defer devices.SetAllForTest()()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testMigCaps := nvcaps.MigCaps{
		nvcaps.MigCap("config"):  nvcaps.MigMinor(1),
		nvcaps.MigCap("monitor"): nvcaps.MigMinor(2),
	}

	testCases := []struct {
		description     string
		rootfs          string
		migCaps         nvcaps.MigCaps
		cap             nvcaps.MigCap
		expectedDevices []Device
		expectedError   bool
	}{
		{
			description:     "nil migCaps returns no devices",
			rootfs:          "rootfs-1",
			migCaps:         nil,
			cap:             nvcaps.MigCap("config"),
			expectedDevices: nil,
		},
		{
			description:   "cap not in migCaps returns error",
			rootfs:        "rootfs-1",
			migCaps:       testMigCaps,
			cap:           nvcaps.MigCap("gpu0/gi0/access"),
			expectedError: true,
		},
		{
			description: "config cap on empty rootfs returns no devices",
			rootfs:      "rootfs-empty",
			migCaps:     testMigCaps,
			cap:         nvcaps.MigCap("config"),
		},
		{
			description: "config cap returns nvidia-cap1",
			rootfs:      "rootfs-1",
			migCaps:     testMigCaps,
			cap:         nvcaps.MigCap("config"),
			expectedDevices: []Device{
				{Path: "/dev/nvidia-caps/nvidia-cap1", HostPath: "/dev/nvidia-caps/nvidia-cap1"},
			},
		},
		{
			description: "monitor cap returns nvidia-cap2",
			rootfs:      "rootfs-1",
			migCaps:     testMigCaps,
			cap:         nvcaps.MigCap("monitor"),
			expectedDevices: []Device{
				{Path: "/dev/nvidia-caps/nvidia-cap2", HostPath: "/dev/nvidia-caps/nvidia-cap2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			devRoot := filepath.Join(lookupRoot, tc.rootfs)
			driver := root.New(root.WithDevRoot(devRoot))

			d, err := newMigCapDiscoverer(logger, driver, tc.migCaps, tc.cap)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got, err := d.Devices()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedDevices, test.StripRoot(got, devRoot))

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.Empty(t, mounts)

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.Empty(t, hooks)

			envVars, err := d.EnvVars()
			require.NoError(t, err)
			require.Empty(t, envVars)
		})
	}
}
