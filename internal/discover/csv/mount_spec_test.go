/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package csv

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMountSpecFromLine(t *testing.T) {
	parseError := fmt.Errorf("failed to parse line")
	unexpectedError := fmt.Errorf("unexpected mount type")

	testCases := []struct {
		line          string
		expectedError error
		expectedValue MountSpec
	}{
		{
			line:          "",
			expectedError: parseError,
		},
		{
			line:          "\t",
			expectedError: parseError,
		},
		{
			line:          ",",
			expectedError: parseError,
		},
		{
			line:          "dev,",
			expectedError: parseError,
		},
		{
			line: "dev ,/a/path",
			expectedValue: MountSpec{
				Path: "/a/path",
				Type: "dev",
			},
		},
		{
			line: "dev ,/a/path,with,commas",
			expectedValue: MountSpec{
				Path: "/a/path,with,commas",
				Type: "dev",
			},
		},
		{
			line:          "not-dev ,/a/path",
			expectedError: unexpectedError,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			target, err := NewMountSpecFromLine(tc.line)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, &tc.expectedValue, target)
		})
	}
}
