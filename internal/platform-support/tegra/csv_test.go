/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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

package tegra

import (
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestNewFromMountSpec(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	locators := map[csv.MountSpecType]lookup.Locator{
		"dev": &lookup.LocatorMock{
			LocateFunc: func(pattern string) ([]string, error) {
				return []string{"/dev/" + pattern}, nil
			},
		},
		"lib": &lookup.LocatorMock{
			LocateFunc: func(pattern string) ([]string, error) {
				return []string{"/lib/" + pattern}, nil
			},
		},
	}

	testCases := []struct {
		description     string
		root            string
		targets         []*csv.MountSpec
		expectedError   error
		expectedDevices []discover.Device
		expectedMounts  []discover.Mount
		expectedHooks   []discover.Hook
	}{
		{
			description:     "empty targets returns None discoverer list",
			expectedDevices: []discover.Device{},
			expectedMounts:  []discover.Mount{},
			expectedHooks:   []discover.Hook{},
		},
		{
			description: "unexpected locator returns error",
			targets: []*csv.MountSpec{
				{
					Type: "foo",
					Path: "bar",
				},
			},
			expectedError: fmt.Errorf("no locator defined for foo"),
		},
		{
			description: "creates discoverers based on type",
			targets: []*csv.MountSpec{
				{
					Type: "dev",
					Path: "dev0",
				},
				{
					Type: "lib",
					Path: "lib0",
				},
				{
					Type: "dev",
					Path: "dev1",
				},
			},
			expectedDevices: []discover.Device{
				{Path: "/dev/dev0", HostPath: "/dev/dev0"},
				{Path: "/dev/dev1", HostPath: "/dev/dev1"},
			},
			expectedMounts: []discover.Mount{
				{Path: "/lib/lib0", HostPath: "/lib/lib0", Options: []string{"ro", "nosuid", "nodev", "bind"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			discoverer, err := newFromMountSpecs(logger, locators, tc.root, tc.targets)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			devices, err := discoverer.Devices()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedDevices, devices)

			mounts, err := discoverer.Mounts()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMounts, mounts)

			hooks, err := discoverer.Hooks()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedHooks, hooks)
		})
	}
}
