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

package lookup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
)

func TestLDCacheLocator(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testDir := t.TempDir()
	symlinkDir := filepath.Join(testDir, "/lib/symlink")
	require.NoError(t, os.MkdirAll(symlinkDir, 0755))

	versionLib := filepath.Join(symlinkDir, "libcuda.so.1.2.3")
	soLink := filepath.Join(symlinkDir, "libcuda.so")
	sonameLink := filepath.Join(symlinkDir, "libcuda.so.1")

	_, err := os.Create(versionLib)
	require.NoError(t, err)
	require.NoError(t, os.Symlink(versionLib, sonameLink))
	require.NoError(t, os.Symlink(sonameLink, soLink))

	lut := newLdcacheLocator(
		WithLogger(logger),
		WithRoot(testDir),
	)

	testCases := []struct {
		description   string
		libname       string
		ldcacheMap    map[string]string
		expected      []string
		expectedError error
	}{
		{
			description: "lib only resolves in LDCache",
			libname:     "libcuda.so",
			ldcacheMap: map[string]string{
				"libcuda.so": "/lib/from/ldcache/libcuda.so.4.5.6",
			},
			expected: []string{"/lib/from/ldcache/libcuda.so.4.5.6"},
		},
		{
			description:   "lib only not in LDCache returns error",
			libname:       "libnotcuda.so",
			expectedError: ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// We override the LDCache with a mock implementation
			l := lut.(*ldcacheLocator)
			l.cache = &ldcache.LDCacheMock{
				LookupFunc: func(strings ...string) ([]string, []string) {
					var result []string
					for _, s := range strings {
						if v, ok := tc.ldcacheMap[s]; ok {
							result = append(result, v)
						}
					}
					return nil, result
				},
			}

			candidates, err := lut.Locate(tc.libname)
			require.ErrorIs(t, err, tc.expectedError)

			var cleanedCandidates []string
			for _, c := range candidates {
				// On MacOS /var and /tmp symlink to /private/var and /private/tmp which is included in the resolved path.
				cleanedCandidates = append(cleanedCandidates, strings.TrimPrefix(c, "/private"))
			}
			require.EqualValues(t, tc.expected, cleanedCandidates)
		})
	}
}

func TestLibraryLocator(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testDir := t.TempDir()
	symlinkDir := filepath.Join(testDir, "/lib/symlink")
	require.NoError(t, os.MkdirAll(symlinkDir, 0755))

	versionLib := filepath.Join(symlinkDir, "libcuda.so.1.2.3")
	soLink := filepath.Join(symlinkDir, "libcuda.so")
	sonameLink := filepath.Join(symlinkDir, "libcuda.so.1")

	f, err := os.Create(versionLib)
	require.NoError(t, err)
	f.Close()
	require.NoError(t, os.Symlink(versionLib, sonameLink))
	require.NoError(t, os.Symlink(sonameLink, soLink))

	// We create a set of symlinks for duplicate resolution
	libTarget1 := filepath.Join(symlinkDir, "libtarget.so.1.2.3")
	source1 := filepath.Join(symlinkDir, "libsource1.so")
	source2 := filepath.Join(symlinkDir, "libsource2.so")

	target1, err := os.Create(libTarget1)
	require.NoError(t, err)
	target1.Close()
	require.NoError(t, os.Symlink(libTarget1, source1))
	require.NoError(t, os.Symlink(source1, source2))

	testCases := []struct {
		description        string
		libname            string
		librarySearchPaths []string
		expected           []string
		expectedError      error
	}{
		{
			description: "slash in path resoves symlink",
			libname:     "/lib/symlink/libcuda.so",
			expected:    []string{filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3")},
		},
		{
			description: "slash in path resoves symlink",
			libname:     "/lib/symlink/libcuda.so.1",
			expected:    []string{filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3")},
		},
		{
			description: "slash in path with pattern resolves symlinks",
			libname:     "/lib/symlink/libcuda.so.*",
			expected:    []string{filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3")},
		},
		{
			description:   "library not found returns error",
			libname:       "/lib/symlink/libnotcuda.so",
			expectedError: ErrNotFound,
		},
		{
			description: "slash in path with pattern resoves symlink",
			libname:     "/lib/symlink/libcuda.so.*.*.*",
			expected:    []string{filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3")},
		},
		{
			description: "symlinks are deduplicated",
			libname:     "/lib/symlink/libsource*.so",
			expected:    []string{filepath.Join(testDir, "/lib/symlink/libtarget.so.1.2.3")},
		},
		{
			description: "pattern resolves to multiple targets",
			libname:     "/lib/symlink/lib*.so.1.2.3",
			expected: []string{
				filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3"),
				filepath.Join(testDir, "/lib/symlink/libtarget.so.1.2.3"),
			},
		},
		{
			description:        "search paths are searched",
			libname:            "lib*.so.1.2.3",
			librarySearchPaths: []string{filepath.Join(testDir, "/lib/symlink")},
			expected: []string{
				filepath.Join(testDir, "/lib/symlink/libcuda.so.1.2.3"),
				filepath.Join(testDir, "/lib/symlink/libtarget.so.1.2.3"),
			},
		},
		{
			description:        "search paths are absolute to root",
			libname:            "lib*.so.1.2.3",
			librarySearchPaths: []string{"/lib/symlink"},
			expectedError:      ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			lut := NewLibraryLocator(
				WithLogger(logger),
				WithRoot(testDir),
				WithSearchPaths(tc.librarySearchPaths...),
			)

			candidates, err := lut.Locate(tc.libname)
			require.ErrorIs(t, err, tc.expectedError)

			var cleanedCandidates []string
			for _, c := range candidates {
				// On MacOS /var and /tmp symlink to /private/var and /private/tmp which is included in the resolved path.
				cleanedCandidates = append(cleanedCandidates, strings.TrimPrefix(c, "/private"))
			}
			require.EqualValues(t, tc.expected, cleanedCandidates)
		})
	}
}
