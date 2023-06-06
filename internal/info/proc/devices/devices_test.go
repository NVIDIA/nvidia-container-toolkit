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

package devices

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNvidiaDevices(t *testing.T) {
	devices := map[Name]Major{
		"nvidia-frontend": 195,
		"nvidia-nvlink":   234,
		"nvidia-caps":     235,
		"nvidia-uvm":      510,
		"nvidia-nvswitch": 511,
	}

	nvidiaDevices := testDevices(devices)
	for name, major := range devices {
		device, exists := nvidiaDevices.Get(name)
		require.True(t, exists, "Unexpected missing device")
		require.Equal(t, device, major, "Unexpected device major")
	}
	_, exists := nvidiaDevices.Get("bogus")
	require.False(t, exists, "Unexpected 'bogus' device found")
}

func TestProcessDeviceFile(t *testing.T) {
	testCases := []struct {
		lines         []string
		expected      devices
		expectedError error
	}{
		{lines: []string{}, expectedError: errNoNvidiaDevices},
		{lines: []string{"Not a valid line:"}, expectedError: errNoNvidiaDevices},
		{lines: []string{"195 nvidia-frontend"}, expected: devices{"nvidia-frontend": 195}},
		{lines: []string{"195 nvidia-frontend", "235 nvidia-caps"}, expected: devices{"nvidia-frontend": 195, "nvidia-caps": 235}},
		{lines: []string{"  195 nvidia-frontend"}, expected: devices{"nvidia-frontend": 195}},
		{lines: []string{"Not a valid line:", "", "195 nvidia-frontend"}, expected: devices{"nvidia-frontend": 195}},
		{lines: []string{"195 not-nvidia-frontend"}, expectedError: errNoNvidiaDevices},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			contents := strings.NewReader(strings.Join(tc.lines, "\n"))
			d, err := nvidiaDeviceFrom(contents)
			require.ErrorIs(t, err, tc.expectedError)

			require.EqualValues(t, tc.expected, d)
		})
	}
}

func TestProcessDeviceFileLine(t *testing.T) {
	testCases := []struct {
		line  string
		name  Name
		major Major
		err   bool
	}{
		{"", "", 0, true},
		{"0", "", 0, true},
		{"notint nvidia-frontend", "", 0, true},
		{"195 nvidia-frontend", "nvidia-frontend", 195, false},
		{"   195 nvidia-frontend", "nvidia-frontend", 195, false},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			name, major, err := processProcDeviceLine(tc.line)

			require.Equal(t, tc.name, name)
			require.Equal(t, tc.major, major)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

		})
	}
}

// testDevices creates a set of test NVIDIA devices
func testDevices(d map[Name]Major) Devices {
	return devices(d)
}
