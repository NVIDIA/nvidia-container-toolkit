/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateMuslPathFileIfRequired(t *testing.T) {
	arch, ok := muslArchs[runtime.GOARCH]
	if !ok {
		t.Skip("musl .path files are not handled on this platform")
	}

	testCases := []struct {
		description string
		// isMusl adds the musl dynamic linker.
		isMusl bool
		// pathFileContents is the contents of the .path file before the update.
		pathFileContents *string
		driverDirs       []string
		systemDirs       []string
		expected         *string
	}{
		{
			description: "glibc container is not modified",
			driverDirs:  []string{"/driver"},
			systemDirs:  []string{"/lib", "/usr/lib"},
		},
		{
			description: "path file is created with the default search path preserved",
			isMusl:      true,
			driverDirs:  []string{"/driver"},
			systemDirs:  []string{"/lib", "/usr/lib"},
			expected:    ptr("/driver\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "driver dirs are prepended to the existing contents",
			isMusl:           true,
			pathFileContents: ptr("/lib:/usr/local/lib:/usr/lib"),
			driverDirs:       []string{"/driver"},
			systemDirs:       []string{"/lib", "/usr/lib"},
			expected:         ptr("/driver\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "dirs that are searched already are not reordered",
			isMusl:           true,
			pathFileContents: ptr("/lib\n/usr/local/lib\n/driver\n"),
			driverDirs:       []string{"/driver"},
			expected:         ptr("/lib\n/usr/local/lib\n/driver\n"),
		},
		{
			description: "the order of the driver dirs is maintained",
			isMusl:      true,
			driverDirs:  []string{"/driver-2", "/driver-1"},
			expected:    ptr("/driver-2\n/driver-1\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "entries separated by colons and newlines are read",
			isMusl:           true,
			pathFileContents: ptr("/lib:/usr/local/lib\n/usr/lib"),
			driverDirs:       []string{"/driver"},
			expected:         ptr("/driver\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description: "system dirs alone create the path file",
			isMusl:      true,
			systemDirs:  []string{"/lib", "/usr/lib"},
			expected:    ptr("/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description: "no dirs leave the container untouched",
			isMusl:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "/etc"), 0o755))
			if tc.pathFileContents != nil {
				require.NoError(t, os.WriteFile(muslPathFile(root, arch), []byte(*tc.pathFileContents), 0o600))
			}
			if tc.isMusl {
				makeMuslLoader(t, root, arch)
			}

			require.NoError(t, createMuslPathFileIfRequired(root, tc.driverDirs, tc.systemDirs))

			requireFileContents(t, muslPathFile(root, arch), tc.expected)
		})
	}
}

func TestIsMusl(t *testing.T) {
	arch, ok := muslArchs[runtime.GOARCH]
	if !ok {
		t.Skip("musl .path files are not handled on this platform")
	}

	t.Run("glibc container", func(t *testing.T) {
		require.False(t, isMusl(t.TempDir(), arch))
	})

	t.Run("musl loader", func(t *testing.T) {
		root := t.TempDir()
		makeMuslLoader(t, root, arch)
		require.True(t, isMusl(root, arch))
	})

	t.Run("alpine release file", func(t *testing.T) {
		root := t.TempDir()
		makeFile(t, filepath.Join(root, "/etc/alpine-release"))
		require.True(t, isMusl(root, arch))
	})
}

// makeMuslLoader creates the musl dynamic linker for the specified
// architecture in the specified root.
func makeMuslLoader(t *testing.T, root string, arch string) {
	t.Helper()

	makeFile(t, muslLoader(root, arch))
}

// requireFileContents checks the contents of the specified file.
// If the expected contents are nil, the file is required to not exist.
func requireFileContents(t *testing.T, path string, expected *string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if expected == nil {
		require.ErrorIs(t, err, os.ErrNotExist)
		return
	}
	require.NoError(t, err)
	require.Equal(t, *expected, string(contents))
}

func ptr[T any](v T) *T {
	return &v
}
