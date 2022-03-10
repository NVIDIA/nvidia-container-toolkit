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

package discover

import (
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestCSVDiscoverer(t *testing.T) {
	logger, logHook := testlog.NewNullLogger()

	testCases := []struct {
		description          string
		input                csvDiscoverer
		expectedMounts       []Mount
		expectedMountsError  error
		expectedDevicesError error
		expectedDevices      []Device
	}{
		{
			description: "dev mounts are empty",
			input: csvDiscoverer{
				mounts: mounts{
					lookup: &lookup.LocatorMock{
						LocateFunc: func(string) ([]string, error) {
							return []string{"located"}, nil
						},
					},
					required: []string{"required"},
				},
				mountType: "dev",
			},
			expectedDevices: []Device{{Path: "located"}},
		},
		{
			description: "dev devices returns error for nil lookup",
			input: csvDiscoverer{
				mountType: "dev",
			},
			expectedDevicesError: fmt.Errorf("no lookup defined"),
		},
		{
			description: "lib devices are empty",
			input: csvDiscoverer{
				mounts: mounts{
					lookup: &lookup.LocatorMock{
						LocateFunc: func(string) ([]string, error) {
							return []string{"located"}, nil
						},
					},
					required: []string{"required"},
				},
				mountType: "lib",
			},
			expectedMounts: []Mount{{Path: "located"}},
		},
		{
			description: "lib mounts returns error for nil lookup",
			input: csvDiscoverer{
				mountType: "lib",
			},
			expectedMountsError: fmt.Errorf("no lookup defined"),
		},
	}

	for _, tc := range testCases {
		logHook.Reset()

		t.Run(tc.description, func(t *testing.T) {
			tc.input.logger = logger

			mounts, err := tc.input.Mounts()
			if tc.expectedMountsError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tc.expectedMounts, mounts)

			devices, err := tc.input.Devices()
			if tc.expectedDevicesError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tc.expectedDevices, devices)
		})
	}
}

func TestNewFromMountSpec(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	locators := map[csv.MountSpecType]lookup.Locator{
		"dev": &lookup.LocatorMock{},
		"lib": &lookup.LocatorMock{},
	}

	testCases := []struct {
		description            string
		targets                []*csv.MountSpec
		expectedError          error
		expectedCSVDiscoverers []*csvDiscoverer
	}{
		{
			description: "empty targets returns empyt list",
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
			expectedCSVDiscoverers: []*csvDiscoverer{
				{
					mountType: "dev",
					mounts: mounts{
						logger:   logger,
						lookup:   locators["dev"],
						required: []string{"dev0", "dev1"},
					},
				},
				{
					mountType: "lib",
					mounts: mounts{
						logger:   logger,
						lookup:   locators["lib"],
						required: []string{"lib0"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			discoverers, err := newFromMountSpecs(logger, locators, tc.targets)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expectedCSVDiscoverers, discoverers)
		})
	}
}
