/*
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

package nvcaps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessMinorsFile(t *testing.T) {
	testCases := []struct {
		lines    []string
		expected MigCaps
	}{
		{[]string{}, MigCaps{}},
		{[]string{"invalidLine"}, MigCaps{}},
		{[]string{"config 1"}, MigCaps{"config": 1}},
		{[]string{"gpu0/gi0/ci0/access 4"}, MigCaps{"gpu0/gi0/ci0/access": 4}},
		{[]string{"config 1", "invalidLine"}, MigCaps{"config": 1}},
		{[]string{"config 1", "gpu0/gi0/ci0/access 4"}, MigCaps{"config": 1, "gpu0/gi0/ci0/access": 4}},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			contents := strings.NewReader(strings.Join(tc.lines, "\n"))
			d := processMinorsFile(contents)
			require.Equal(t, tc.expected, d)
		})
	}
}

func TestProcessMigMinorsLine(t *testing.T) {
	testCases := []struct {
		line  string
		cap   MigCap
		minor MigMinor
		err   bool
	}{
		{"config 1", "config", 1, false},
		{"monitor 2", "monitor", 2, false},
		{"gpu0/gi0/access 3", "gpu0/gi0/access", 3, false},
		{"gpu0/gi0/ci0/access 4", "gpu0/gi0/ci0/access", 4, false},
		{"notconfig 99", "", 0, true},
		{"config notanint", "", 0, true},
		{"", "", 0, true},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			cap, minor, err := processMigMinorsLine(tc.line)

			require.Equal(t, tc.cap, cap)
			require.Equal(t, tc.minor, minor)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMigCapProcPaths(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"config", "/proc/driver/nvidia/capabilities/mig/config"},
		{"monitor", "/proc/driver/nvidia/capabilities/mig/monitor"},
		{"gpu0/gi0/access", "/proc/driver/nvidia/capabilities/gpu0/mig/gi0/access"},
		{"gpu0/gi0/ci0/access", "/proc/driver/nvidia/capabilities/gpu0/mig/gi0/ci0/access"},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			m := MigCap(tc.input)
			require.Equal(t, tc.expected, m.ProcPath())
		})
	}
}

func TestMigMinorDevicePath(t *testing.T) {
	m := MigMinor(0)
	require.Equal(t, "/dev/nvidia-caps/nvidia-cap0", m.DevicePath())
}
