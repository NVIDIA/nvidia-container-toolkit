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

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	testCases := []struct {
		args              []string
		expectedRemaining []string
		expectedRoot      string
		expectedError     error
	}{
		{
			args:              []string{},
			expectedRemaining: []string{},
			expectedRoot:      "",
			expectedError:     nil,
		},
		{
			args:              []string{"app"},
			expectedRemaining: []string{"app"},
		},
		{
			args:              []string{"app", "root"},
			expectedRemaining: []string{"app"},
			expectedRoot:      "root",
		},
		{
			args:              []string{"app", "--flag"},
			expectedRemaining: []string{"app", "--flag"},
		},
		{
			args:              []string{"app", "root", "--flag"},
			expectedRemaining: []string{"app", "--flag"},
			expectedRoot:      "root",
		},
		{
			args:          []string{"app", "root", "not-root", "--flag"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root]"),
		},
		{
			args:          []string{"app", "root", "not-root"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root]"),
		},
		{
			args:          []string{"app", "root", "not-root", "also"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root also]"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			remaining, root, err := ParseArgs(tc.args)
			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			require.ElementsMatch(t, tc.expectedRemaining, remaining)
			require.Equal(t, tc.expectedRoot, root)
		})
	}
}
