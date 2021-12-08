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

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDriverCapabilitiesIntersection(t *testing.T) {
	testCases := []struct {
		capabilities          DriverCapabilities
		supportedCapabilities DriverCapabilities
		expectedIntersection  DriverCapabilities
	}{
		{
			capabilities:          none,
			supportedCapabilities: none,
			expectedIntersection:  none,
		},
		{
			capabilities:          all,
			supportedCapabilities: none,
			expectedIntersection:  none,
		},
		{
			capabilities:          all,
			supportedCapabilities: allDriverCapabilities,
			expectedIntersection:  allDriverCapabilities,
		},
		{
			capabilities:          allDriverCapabilities,
			supportedCapabilities: all,
			expectedIntersection:  allDriverCapabilities,
		},
		{
			capabilities:          none,
			supportedCapabilities: all,
			expectedIntersection:  none,
		},
		{
			capabilities:          none,
			supportedCapabilities: DriverCapabilities("cap1"),
			expectedIntersection:  none,
		},
		{
			capabilities:          DriverCapabilities("cap0,cap1"),
			supportedCapabilities: DriverCapabilities("cap1,cap0"),
			expectedIntersection:  DriverCapabilities("cap0,cap1"),
		},
		{
			capabilities:          defaultDriverCapabilities,
			supportedCapabilities: allDriverCapabilities,
			expectedIntersection:  defaultDriverCapabilities,
		},
		{
			capabilities:          DriverCapabilities("compute,compat32,graphics,utility,video,display"),
			supportedCapabilities: DriverCapabilities("compute,compat32,graphics,utility,video,display,ngx"),
			expectedIntersection:  DriverCapabilities("compute,compat32,graphics,utility,video,display"),
		},
		{
			capabilities:          DriverCapabilities("cap1"),
			supportedCapabilities: none,
			expectedIntersection:  none,
		},
		{
			capabilities:          DriverCapabilities("compute,compat32,graphics,utility,video,display,ngx"),
			supportedCapabilities: DriverCapabilities("compute,compat32,graphics,utility,video,display"),
			expectedIntersection:  DriverCapabilities("compute,compat32,graphics,utility,video,display"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			intersection := tc.supportedCapabilities.Intersection(tc.capabilities)
			require.EqualValues(t, tc.expectedIntersection, intersection)
		})
	}
}

func TestDriverCapabilitiesList(t *testing.T) {
	testCases := []struct {
		capabilities DriverCapabilities
		expected     []string
	}{
		{
			capabilities: DriverCapabilities(""),
		},
		{
			capabilities: DriverCapabilities("  "),
		},
		{
			capabilities: DriverCapabilities(","),
		},
		{
			capabilities: DriverCapabilities(",cap"),
			expected:     []string{"cap"},
		},
		{
			capabilities: DriverCapabilities("cap,"),
			expected:     []string{"cap"},
		},
		{
			capabilities: DriverCapabilities("cap0,,cap1"),
			expected:     []string{"cap0", "cap1"},
		},
		{
			capabilities: DriverCapabilities("cap1,cap0,cap3"),
			expected:     []string{"cap1", "cap0", "cap3"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			require.EqualValues(t, tc.expected, tc.capabilities.list())
		})
	}
}
