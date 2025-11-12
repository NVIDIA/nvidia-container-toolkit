/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package ldconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterDirectories(t *testing.T) {
	const topLevelConf = "TOPLEVEL.conf"

	testCases := []struct {
		description string
		confs       map[string]string // map[filename]content, must have topLevelConf key
		input       []string
		expected    []string
	}{
		{
			description: "all filtered",
			confs: map[string]string{
				topLevelConf: `
# some comment
/tmp/libdir1
/tmp/libdir2
`,
			},
			input:    []string{"/tmp/libdir1", "/tmp/libdir2"},
			expected: nil,
		},
		{
			description: "partially filtered",
			confs: map[string]string{
				topLevelConf: `
/tmp/libdir1
`,
			},
			input:    []string{"/tmp/libdir1", "/tmp/libdir2"},
			expected: []string{"/tmp/libdir2"},
		},
		{
			description: "none filtered",
			confs: map[string]string{
				topLevelConf: `
# empty config
`,
			},
			input:    []string{"/tmp/libdir1", "/tmp/libdir2"},
			expected: []string{"/tmp/libdir1", "/tmp/libdir2"},
		},
		{
			description: "filter with include and comments",
			confs: map[string]string{
				topLevelConf: `
# comment
/tmp/libdir1
include /nonexistent/pattern*
`,
			},
			input:    []string{"/tmp/libdir1", "/tmp/libdir2"},
			expected: []string{"/tmp/libdir2"},
		},
		{
			description: "include directive picks up more dirs to filter",
			confs: map[string]string{
				topLevelConf: `
# top-level
include INCLUDED_PATTERN*
/tmp/libdir3
`,
				"INCLUDED_PATTERN0.conf": `
/tmp/libdir2
# another comment
/tmp/libdir4
`,
				"INCLUDED_PATTERN1.conf": `
/tmp/libdir1
`,
			},
			input:    []string{"/tmp/libdir1", "/tmp/libdir2", "/tmp/libdir3", "/tmp/libdir4", "/tmp/libdir5"},
			expected: []string{"/tmp/libdir5"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Prepare file contents, adjusting include globs to be absolute and unique within tmpDir
			for name, content := range tc.confs {
				if name == topLevelConf && len(tc.confs) > 1 {
					content = strings.ReplaceAll(content, "include INCLUDED_PATTERN*", "include "+tmpDir+"/INCLUDED_PATTERN*")
				}
				err := os.WriteFile(tmpDir+"/"+name, []byte(content), 0600)
				require.NoError(t, err)
			}

			topLevelConfPath := tmpDir + "/" + topLevelConf
			l := &Ldconfig{
				isDebianLikeContainer: true,
			}
			filtered, err := l.filterDirectories(topLevelConfPath, tc.input...)

			require.NoError(t, err)
			require.Equal(t, tc.expected, filtered)
		})
	}
}
