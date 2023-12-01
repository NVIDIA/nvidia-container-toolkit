/*
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
*/

package discover

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"

	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestMountsReturnsEmptyDevices(t *testing.T) {
	d := mounts{}
	devices, err := d.Devices()

	require.NoError(t, err)
	require.Empty(t, devices)
}

func TestMounts(t *testing.T) {

	mountOptions := []string{
		"ro",
		"nosuid",
		"nodev",
		"bind",
	}

	logger, logHook := testlog.NewNullLogger()

	testCases := []struct {
		description    string
		expectedError  error
		expectedMounts []Mount
		input          *mounts
	}{
		{
			description:   "nill lookup returns error",
			expectedError: fmt.Errorf("no lookup defined"),
			input:         &mounts{},
		},
		{
			description:   "empty required returns no mounts",
			expectedError: nil,
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(string) ([]string, error) {
						return []string{"located"}, nil
					},
				},
			},
		},
		{
			description:   "required returns located",
			expectedError: nil,
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(string) ([]string, error) {
						return []string{"located"}, nil
					},
				},
				required: []string{"required"},
			},
			expectedMounts: []Mount{{Path: "located", HostPath: "located", Options: mountOptions}},
		},
		{
			description:   "mounts removes located duplicates",
			expectedError: nil,
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(string) ([]string, error) {
						return []string{"located"}, nil
					},
				},
				required: []string{"required0", "required1"},
			},
			expectedMounts: []Mount{{Path: "located", HostPath: "located", Options: mountOptions}},
		},
		{
			description: "mounts skips located errors",
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(s string) ([]string, error) {
						if s == "error" {
							return nil, fmt.Errorf(s)
						}
						return []string{s}, nil
					},
				},
				required: []string{"required0", "error", "required1"},
			},
			expectedMounts: []Mount{{Path: "required0", HostPath: "required0", Options: mountOptions}, {Path: "required1", HostPath: "required1", Options: mountOptions}},
		},
		{
			description: "mounts skips unlocated",
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(s string) ([]string, error) {
						if s == "empty" {
							return nil, nil
						}
						return []string{s}, nil
					},
				},
				required: []string{"required0", "empty", "required1"},
			},
			expectedMounts: []Mount{{Path: "required0", HostPath: "required0", Options: mountOptions}, {Path: "required1", HostPath: "required1", Options: mountOptions}},
		},
		{
			description: "mounts adds multiple",
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(s string) ([]string, error) {
						if s == "multiple" {
							return []string{"multiple0", "multiple1"}, nil
						}
						return []string{s}, nil
					},
				},
				required: []string{"required0", "multiple", "required1"},
			},
			expectedMounts: []Mount{
				{Path: "required0", HostPath: "required0", Options: mountOptions},
				{Path: "multiple0", HostPath: "multiple0", Options: mountOptions},
				{Path: "multiple1", HostPath: "multiple1", Options: mountOptions},
				{Path: "required1", HostPath: "required1", Options: mountOptions},
			},
		},
		{
			description: "mounts uses relative path",
			input: &mounts{
				lookup: &lookup.LocatorMock{
					LocateFunc: func(s string) ([]string, error) {
						return []string{"/some/root/located"}, nil
					},
				},
				root:     "/some/root",
				required: []string{"required0", "multiple", "required1"},
			},
			expectedMounts: []Mount{
				{Path: "/located", HostPath: "/some/root/located", Options: mountOptions},
			},
		},
	}

	for _, tc := range testCases {
		logHook.Reset()
		t.Run(tc.description, func(t *testing.T) {
			tc.input.logger = logger
			mounts, err := tc.input.Mounts()

			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tc.expectedMounts, mounts)

			// We check that the mock is called for each element of required
			if tc.input.lookup != nil {
				mock := tc.input.lookup.(*lookup.LocatorMock)
				require.Len(t, mock.LocateCalls(), len(tc.input.required))
				var args []string
				for _, c := range mock.LocateCalls() {
					args = append(args, c.S)
				}
				require.EqualValues(t, args, tc.input.required)
			}
		})
	}
}
