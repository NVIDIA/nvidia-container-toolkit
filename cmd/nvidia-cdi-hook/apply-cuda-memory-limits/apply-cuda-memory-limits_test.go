/**
# SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package cudamemorylimits

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMebiBytes(t *testing.T) {
	testCases := []struct {
		description string
		value       string
		expected    uint64
		expectError bool
	}{
		{
			description: "valid value",
			value:       "1024",
			expected:    1024,
		},
		{
			description: "zero",
			value:       "0",
			expected:    0,
		},
		{
			description: "largest value convertible to bytes",
			value:       "17592186044415",
			expected:    maxMebiBytes,
		},
		{
			description: "one more than the largest convertible value",
			value:       "17592186044416",
			expectError: true,
		},
		{
			description: "max uint64",
			value:       "18446744073709551615",
			expectError: true,
		},
		{
			description: "value exceeding uint64",
			value:       "18446744073709551616",
			expectError: true,
		},
		{
			description: "non-numeric value",
			value:       "12abc",
			expectError: true,
		},
		{
			description: "empty value",
			value:       "",
			expectError: true,
		},
		{
			description: "negative value",
			value:       "-1",
			expectError: true,
		},
		{
			description: "leading whitespace",
			value:       " 1024",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mebiBytes, err := parseMebiBytes(tc.value)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, mebiBytes)
			// The value must be convertible to bytes without overflowing.
			require.Equal(t, tc.expected, mebiBytes*mebiByteMultiplier/mebiByteMultiplier)
		})
	}
}
