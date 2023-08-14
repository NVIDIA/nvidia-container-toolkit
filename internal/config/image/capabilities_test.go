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

package image

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
			capabilities:          driverCapabilitiesNone,
			supportedCapabilities: driverCapabilitiesNone,
			expectedIntersection:  driverCapabilitiesNone,
		},
		{
			capabilities:          driverCapabilitiesAll,
			supportedCapabilities: driverCapabilitiesNone,
			expectedIntersection:  driverCapabilitiesNone,
		},
		{
			capabilities:          driverCapabilitiesAll,
			supportedCapabilities: SupportedDriverCapabilities,
			expectedIntersection:  SupportedDriverCapabilities,
		},
		{
			capabilities:          SupportedDriverCapabilities,
			supportedCapabilities: driverCapabilitiesAll,
			expectedIntersection:  SupportedDriverCapabilities,
		},
		{
			capabilities:          driverCapabilitiesNone,
			supportedCapabilities: driverCapabilitiesAll,
			expectedIntersection:  driverCapabilitiesNone,
		},
		{
			capabilities:          driverCapabilitiesNone,
			supportedCapabilities: NewDriverCapabilities("cap1"),
			expectedIntersection:  driverCapabilitiesNone,
		},
		{
			capabilities:          NewDriverCapabilities("cap0,cap1"),
			supportedCapabilities: NewDriverCapabilities("cap1,cap0"),
			expectedIntersection:  NewDriverCapabilities("cap0,cap1"),
		},
		{
			capabilities:          DefaultDriverCapabilities,
			supportedCapabilities: SupportedDriverCapabilities,
			expectedIntersection:  DefaultDriverCapabilities,
		},
		{
			capabilities:          NewDriverCapabilities("compute,compat32,graphics,utility,video,display"),
			supportedCapabilities: NewDriverCapabilities("compute,compat32,graphics,utility,video,display,ngx"),
			expectedIntersection:  NewDriverCapabilities("compute,compat32,graphics,utility,video,display"),
		},
		{
			capabilities:          NewDriverCapabilities("cap1"),
			supportedCapabilities: driverCapabilitiesNone,
			expectedIntersection:  driverCapabilitiesNone,
		},
		{
			capabilities:          NewDriverCapabilities("compute,compat32,graphics,utility,video,display,ngx"),
			supportedCapabilities: NewDriverCapabilities("compute,compat32,graphics,utility,video,display"),
			expectedIntersection:  NewDriverCapabilities("compute,compat32,graphics,utility,video,display"),
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
			capabilities: NewDriverCapabilities(""),
		},
		{
			capabilities: NewDriverCapabilities("  "),
		},
		{
			capabilities: NewDriverCapabilities(","),
		},
		{
			capabilities: NewDriverCapabilities(",cap"),
			expected:     []string{"cap"},
		},
		{
			capabilities: NewDriverCapabilities("cap,"),
			expected:     []string{"cap"},
		},
		{
			capabilities: NewDriverCapabilities("cap0,,cap1"),
			expected:     []string{"cap0", "cap1"},
		},
		{
			capabilities: NewDriverCapabilities("cap1,cap0,cap3"),
			expected:     []string{"cap0", "cap1", "cap3"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			require.EqualValues(t, tc.expected, tc.capabilities.List())
		})
	}
}
