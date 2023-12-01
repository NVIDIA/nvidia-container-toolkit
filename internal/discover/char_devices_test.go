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

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

func TestCharDevices(t *testing.T) {
	logger, logHook := testlog.NewNullLogger()

	testCases := []struct {
		description          string
		input                *charDevices
		expectedMounts       []Mount
		expectedMountsError  error
		expectedDevicesError error
		expectedDevices      []Device
	}{
		{
			description: "dev mounts are empty",
			input: (*charDevices)(
				&mounts{
					lookup: &lookup.LocatorMock{
						LocateFunc: func(string) ([]string, error) {
							return []string{"located"}, nil
						},
					},
					required: []string{"required"},
				},
			),
			expectedDevices: []Device{{Path: "located", HostPath: "located"}},
		},
		{
			description:          "dev devices returns error for nil lookup",
			input:                &charDevices{},
			expectedDevicesError: fmt.Errorf("no lookup defined"),
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
