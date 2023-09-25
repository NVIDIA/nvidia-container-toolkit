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

package tegra

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIgnorePatterns(t *testing.T) {
	testCases := []struct {
		description   string
		blockedFilter []string
		input         []string
		expected      []string
	}{
		{
			description: "nil slice",
			input:       []string{"something", "somethingelse"},
			expected:    []string{"something", "somethingelse"},
		},
		{
			description:   "match libraries full path and so symlinks using globs",
			blockedFilter: []string{"*.so", "*.so.[0-9]"},
			input:         []string{"/foo/bar/libsomething.so", "libsometing.so", "libsometing.so.1", "libsometing.so.1.2.3"},
			expected:      []string{"/foo/bar/libsomething.so", "libsometing.so.1.2.3"},
		},
		{
			description:   "match libraries full path and so symlinks using globs with any path prefix",
			blockedFilter: []string{"**/*.so", "**/*.so.[0-9]"},
			input:         []string{"/foo/bar/libsomething.so", "libsometing.so", "libsometing.so.1", "libsometing.so.1.2.3"},
			expected:      []string{"libsometing.so.1.2.3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			filtered := ignoreMountSpecPatterns(tc.blockedFilter).Apply(tc.input...)
			require.ElementsMatch(t, tc.expected, filtered)
		})
	}
}
