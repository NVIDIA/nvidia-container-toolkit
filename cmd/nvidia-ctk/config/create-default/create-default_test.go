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

package defaultsubcommand

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixComment(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "# comment",
			expected: "#comment",
		},
		{
			input:    " #comment",
			expected: "#comment",
		},
		{
			input:    " # comment",
			expected: "#comment",
		},
		{
			input: strings.Join([]string{
				"some",
				"# comment",
				" # comment",
				" #comment",
				"other"}, "\n"),
			expected: strings.Join([]string{
				"some",
				"#comment",
				"#comment",
				"#comment",
				"other"}, "\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, _ := fixComments([]byte(tc.input))
			require.Equal(t, tc.expected, string(actual))
		})
	}
}

func TestGetFormattedConfig(t *testing.T) {
	expectedLines := []string{
		"#no-cgroups = false",
		"#debug = \"/var/log/nvidia-container-toolkit.log\"",
		"#debug = \"/var/log/nvidia-container-runtime.log\"",
	}

	opts := &options{}
	contents, err := opts.getFormattedConfig()
	require.NoError(t, err)
	lines := strings.Split(string(contents), "\n")

	for _, line := range expectedLines {
		require.Contains(t, lines, line)
	}
}
