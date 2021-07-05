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

package proc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNvidiaDevices(t *testing.T) {
	devices := []Device{
		{"nvidia-frontend", 195},
		{"nvidia-nvlink", 234},
		{"nvidia-caps", 235},
		{"nvidia-uvm", 510},
		{"nvidia-nvswitch", 511},
	}

	nvidiaDevices := NewMockNvidiaDevices(devices...)
	for _, d := range devices {
		device, exists := nvidiaDevices.Get(d.Name)
		require.True(t, exists, "Unexpected missing device")
		require.Equal(t, device.Name, d.Name, "Unexpected device name")
		require.Equal(t, device.Major, d.Major, "Unexpected device major")
	}
	_, exists := nvidiaDevices.Get("bogus")
	require.False(t, exists, "Unexpected 'bogus' device found")
}

func TestProcessDeviceFile(t *testing.T) {
	testCases := []struct {
		lines    []string
		expected []Device
	}{
		{[]string{}, []Device{}},
		{[]string{"Not a valid line:"}, []Device{}},
		{[]string{"195 nvidia-frontend"}, []Device{{"nvidia-frontend", 195}}},
		{[]string{"195 nvidia-frontend", "235 nvidia-caps"}, []Device{{"nvidia-frontend", 195}, {"nvidia-caps", 235}}},
		{[]string{"  195 nvidia-frontend"}, []Device{{"nvidia-frontend", 195}}},
		{[]string{"Not a valid line:", "", "195 nvidia-frontend"}, []Device{{"nvidia-frontend", 195}}},
		{[]string{"195 not-nvidia-frontend"}, []Device{}},
	}
	for _, tc := range testCases {
		contents := strings.NewReader(strings.Join(tc.lines, "\n"))
		d := processDeviceFile(contents)
		require.Equalf(t, NewMockNvidiaDevices(tc.expected...), d, "testCase: %v", tc)
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

	for _, tc := range testCases {
		name, major, err := processProcDeviceLine(tc.line)

		require.Equal(t, tc.name, name)
		require.Equal(t, tc.major, major)
		if tc.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

	}
}
