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
	perDriverDeviceMaps := map[string]map[string]int{
		"pre550": {
			"nvidia-frontend": 195,
			"nvidia-nvlink":   234,
			"nvidia-caps":     235,
			"nvidia-uvm":      510,
			"nvidia-nvswitch": 511,
		},
		"post550": {
			"nvidia":          195,
			"nvidia-nvlink":   234,
			"nvidia-caps":     235,
			"nvidia-uvm":      510,
			"nvidia-nvswitch": 511,
		},
	}

	for k, devices := range perDriverDeviceMaps {
		nvidiaDevices := New(WithDeviceToMajor(devices))
		t.Run(k, func(t *testing.T) {
			// Each of the expected devices needs to exist.
			for name, major := range devices {
				device, exists := nvidiaDevices.Get(Name(name))
				require.True(t, exists)
				require.Equal(t, device, Major(major))
			}
			// An unexpected device cannot exist
			_, exists := nvidiaDevices.Get("bogus")
			require.False(t, exists)

			// Regardles of the driver version, the nvidia and nvidia-frontend
			// names are supported and have the same value.
			nvidia, exists := nvidiaDevices.Get(NVIDIAGPU)
			require.True(t, exists)
			nvidiaFrontend, exists := nvidiaDevices.Get(NVIDIAFrontend)
			require.True(t, exists)
			require.Equal(t, nvidia, nvidiaFrontend)
		})

	}
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
		{lines: []string{"195 nvidia"}, expected: devices{"nvidia": 195}},
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

			if tc.expectedError == nil {
				require.EqualValues(t, tc.expected, d.(devices))
			}

		})
	}
}

func TestProcessDeviceFileLine(t *testing.T) {
	testCases := []struct {
		line  string
		name  string
		major int
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
