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
	"path/filepath"
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

func TestCreateLdsoconfdFile(t *testing.T) {
	testCases := []struct {
		description     string
		pattern         string
		dirs            []string
		expectedContent []string
	}{
		{
			description:     "empty directories",
			pattern:         "test-*.conf",
			dirs:            []string{},
			expectedContent: nil,
		},
		{
			description: "single directory",
			pattern:     "test-*.conf",
			dirs:        []string{"/usr/local/lib"},
			expectedContent: []string{
				"/usr/local/lib",
			},
		},
		{
			description: "multiple directories",
			pattern:     "test-*.conf",
			dirs:        []string{"/usr/local/lib", "/opt/lib", "/usr/lib64"},
			expectedContent: []string{
				"/usr/local/lib",
				"/opt/lib",
				"/usr/lib64",
			},
		},
		{
			description: "duplicate directories",
			pattern:     "test-*.conf",
			dirs:        []string{"/usr/local/lib", "/opt/lib", "/usr/local/lib"},
			expectedContent: []string{
				"/usr/local/lib",
				"/opt/lib",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()

			err := createLdsoconfdFile(tmpDir, tc.pattern, tc.dirs...)
			require.NoError(t, err)

			if len(tc.expectedContent) == 0 {
				entries, err := os.ReadDir(tmpDir)
				require.NoError(t, err)
				require.Empty(t, entries)
				return
			}

			entries, err := os.ReadDir(tmpDir)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			createdFile := filepath.Join(tmpDir, entries[0].Name())

			info, err := os.Stat(createdFile)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0644), info.Mode().Perm())

			content, err := os.ReadFile(createdFile)
			require.NoError(t, err)
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			require.Equal(t, tc.expectedContent, lines)
		})
	}
}

func TestEnsureLdsoconfFile(t *testing.T) {
	testCases := []struct {
		description     string
		existingContent string
		ldsoconfdDir    string
		expectCreation  bool
		expectedContent string
	}{
		{
			description:     "creates file when none exists",
			existingContent: "",
			ldsoconfdDir:    "/custom/ld.so.conf.d",
			expectCreation:  true,
			expectedContent: "include /custom/ld.so.conf.d/*.conf\n",
		},
		{
			description:     "does not modify existing file",
			existingContent: "# custom config\n/usr/local/lib\n",
			ldsoconfdDir:    "/etc/ld.so.conf.d",
			expectCreation:  false,
			expectedContent: "# custom config\n/usr/local/lib\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()
			confFilePath := filepath.Join(tmpDir, "ld.so.conf")

			if tc.existingContent != "" {
				err := os.WriteFile(confFilePath, []byte(tc.existingContent), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			err := ensureLdsoconfFile(confFilePath, tc.ldsoconfdDir)
			require.NoError(t, err)

			info, err := os.Stat(confFilePath)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0644), info.Mode().Perm())

			content, err := os.ReadFile(confFilePath)
			require.NoError(t, err)
			require.Equal(t, tc.expectedContent, string(content))
		})
	}
}

func TestProcessLdsoconfFile(t *testing.T) {
	testCases := []struct {
		description string
		// files are created relative to a temporary directory. The entry
		// "ld.so.conf" is the file that is read.
		files               map[string]string
		expectedDirectories []string
		expectedIncludes    []string
	}{
		{
			description: "a comment at the end of a line is not part of the directory",
			files: map[string]string{
				"ld.so.conf": "/usr/local/lib # vendor libraries\n/opt/lib\t# and these\n",
			},
			expectedDirectories: []string{"/usr/local/lib", "/opt/lib"},
		},
		{
			description: "a comment at the end of an include is not part of the pattern",
			files: map[string]string{
				"ld.so.conf":          "include ld.so.conf.d/*.conf # the distro's\n",
				"ld.so.conf.d/a.conf": "/dir-a\n",
			},
			expectedIncludes: []string{"ld.so.conf.d/a.conf"},
		},
		{
			description: "a line that is only a comment is ignored",
			files: map[string]string{
				"ld.so.conf": "# nothing here\n   # nor here\n/dir-a\n",
			},
			expectedDirectories: []string{"/dir-a"},
		},
		{
			description: "an include separated by a tab is a directive",
			files: map[string]string{
				"ld.so.conf":          "include\tld.so.conf.d/*.conf\n",
				"ld.so.conf.d/a.conf": "/dir-a\n",
			},
			expectedIncludes: []string{"ld.so.conf.d/a.conf"},
		},
		{
			description: "every pattern of an include is used",
			files: map[string]string{
				"ld.so.conf":  "include first.conf \t second.conf  third.conf\n",
				"first.conf":  "/dir-a\n",
				"second.conf": "/dir-b\n",
				"third.conf":  "/dir-c\n",
			},
			expectedIncludes: []string{"first.conf", "second.conf", "third.conf"},
		},
		{
			description: "a relative include resolves against the directory of the file",
			files: map[string]string{
				"etc/ld.so.conf":          "include ld.so.conf.d/*.conf\n",
				"etc/ld.so.conf.d/a.conf": "/dir-a\n",
			},
			expectedIncludes: []string{"etc/ld.so.conf.d/a.conf"},
		},
		{
			description: "an absolute include is used as it is",
			files: map[string]string{
				"ld.so.conf": "include {{TMPDIR}}/other.conf\n",
				"other.conf": "/dir-a\n",
			},
			expectedIncludes: []string{"other.conf"},
		},
		{
			description: "a hwcap directive is not a directory",
			files: map[string]string{
				"ld.so.conf": "hwcap 1 nosegneg\nHWCAP 0 something\n/dir-a\n",
			},
			expectedDirectories: []string{"/dir-a"},
		},
		{
			description: "a line that only looks like a directive is a directory",
			files: map[string]string{
				"ld.so.conf": "include\n/includes\n/hwcapsomething\n",
			},
			expectedDirectories: []string{"include", "/includes", "/hwcapsomething"},
		},
		{
			description: "an include that matches nothing is not an error",
			files: map[string]string{
				"ld.so.conf": "include ld.so.conf.d/*.conf\n/dir-a\n",
			},
			expectedDirectories: []string{"/dir-a"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()

			var configFile string
			for name, contents := range tc.files {
				path := filepath.Join(tmpDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				contents = strings.ReplaceAll(contents, "{{TMPDIR}}", tmpDir)
				require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
				if filepath.Base(name) == "ld.so.conf" {
					configFile = path
				}
			}
			require.NotEmpty(t, configFile)

			directories, includes, err := processLdsoconfFile(configFile)
			require.NoError(t, err)

			var expectedIncludes []string
			for _, include := range tc.expectedIncludes {
				expectedIncludes = append(expectedIncludes, filepath.Join(tmpDir, include))
			}

			require.EqualValues(t, tc.expectedDirectories, directories)
			require.EqualValues(t, expectedIncludes, includes)
		})
	}
}
