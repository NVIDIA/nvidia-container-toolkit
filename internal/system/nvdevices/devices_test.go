/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package nvdevices

import (
	"errors"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
)

func TestCreateControlDevices(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	nvidiaDevices := &devices.DevicesMock{
		GetFunc: func(name devices.Name) (devices.Major, bool) {
			devices := map[devices.Name]devices.Major{
				"nvidia-frontend": 195,
				"nvidia-uvm":      243,
			}
			return devices[name], true
		},
	}

	// nvidiaDevices550 represents the device map from the nvidia gpu drivers >= 550.40.x
	nvidiaDevices550 := &devices.DevicesMock{
		GetFunc: func(name devices.Name) (devices.Major, bool) {
			devices := map[devices.Name]devices.Major{
				"nvidia":     195,
				"nvidia-uvm": 243,
			}
			d, ok := devices[name]
			return d, ok
		},
	}

	mknodeError := errors.New("mknode error")

	testCases := []struct {
		description   string
		root          string
		devices       devices.Devices
		mknodeError   error
		hasError      bool
		expectedError error
		expectedCalls []struct {
			S  string
			N1 int
			N2 int
		}
	}{
		{
			description: "no root specified",
			root:        "",
			devices:     nvidiaDevices,
			mknodeError: nil,
			hasError:    false,
			expectedCalls: []struct {
				S  string
				N1 int
				N2 int
			}{
				{"/dev/nvidiactl", 195, 255},
				{"/dev/nvidia-modeset", 195, 254},
				{"/dev/nvidia-uvm", 243, 0},
				{"/dev/nvidia-uvm-tools", 243, 1},
			},
		},
		{
			description: "some root specified",
			root:        "/some/root",
			devices:     nvidiaDevices,
			hasError:    false,
			mknodeError: nil,
			expectedCalls: []struct {
				S  string
				N1 int
				N2 int
			}{
				{"/some/root/dev/nvidiactl", 195, 255},
				{"/some/root/dev/nvidia-modeset", 195, 254},
				{"/some/root/dev/nvidia-uvm", 243, 0},
				{"/some/root/dev/nvidia-uvm-tools", 243, 1},
			},
		},
		{
			description:   "mknod error returns error",
			devices:       nvidiaDevices,
			hasError:      true,
			mknodeError:   mknodeError,
			expectedError: mknodeError,
			// We expect the first call to this to fail, and the rest to be skipped
			expectedCalls: []struct {
				S  string
				N1 int
				N2 int
			}{
				{"/dev/nvidiactl", 195, 255},
			},
		},
		{
			description: "missing major returns error",
			devices: &devices.DevicesMock{
				GetFunc: func(name devices.Name) (devices.Major, bool) {
					return 0, false
				},
			},
			hasError:      true,
			expectedError: errInvalidDeviceNode,
		},
		{
			description:   "nvidia device renamed from nvidia-frontend to nvidia",
			devices:       nvidiaDevices550,
			hasError:      true,
			expectedError: errors.New("failed to create device node nvidiactl"),
			expectedCalls: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mknode := &mknoderMock{
				MknodeFunc: func(string, int, int) error {
					return tc.mknodeError
				},
			}

			d, _ := New(
				WithLogger(logger),
				WithDevRoot(tc.root),
				WithDevices(tc.devices),
			)
			d.mknoder = mknode

			err := d.CreateNVIDIAControlDevices()
			if tc.hasError {
				require.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				require.Nil(t, err)
			}
			require.EqualValues(t, tc.expectedCalls, mknode.MknodeCalls())
		})
	}
}
