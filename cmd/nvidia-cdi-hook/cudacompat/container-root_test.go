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

package cudacompat

import (
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkdirAll(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]string
		path        string
		expectedErr bool
		skipGOOS    []string
	}{
		{
			description: "empty path",
			path:        "",
			expectedErr: false,
		},
		{
			description: "single dir",
			path:        "foo",
			expectedErr: false,
		},
		{
			description: "nested dir",
			path:        "/path/to/foo",
			expectedErr: false,
		},
		{
			description: "relative symlinks are followed in root",
			contents: map[string]string{
				"/x/y":     "",
				"/path/to": "symlink=../x",
			},
			path:        "/path/to/foo",
			expectedErr: false,
		},
		{
			description: "absolute symlinks are followed in root",
			contents: map[string]string{
				"/x/y":     "",
				"/path/to": "symlink=/x",
			},
			path:        "/path/to/foo",
			expectedErr: false,
			skipGOOS:    []string{"darwin"},
		},
		{
			description: "fails if symlink points to invalid path",
			contents: map[string]string{
				"/path/to": "symlink=../x",
			},
			path:        "/path/to/foo",
			expectedErr: true,
			skipGOOS:    []string{"darwin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if slices.Contains(tc.skipGOOS, runtime.GOOS) {
				t.Skipf("test not supported on %s", runtime.GOOS)
			}

			containerRootDir := t.TempDir()
			for name, contents := range tc.contents {
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			root, _ := newRoot(containerRootDir)
			err := root.MkdirAll(tc.path, 0755)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			f, err := root.Open(tc.path)
			require.NoError(t, err)
			info, err := f.Stat()
			require.NoError(t, err)
			require.True(t, info.IsDir())
		})
	}
}

func TestOpen(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]string
		path        string
		expectedErr bool
		skipGOOS    []string
	}{
		{
			description: "empty path",
			path:        "",
		},
		{
			description: "file does not exist",
			path:        "foo",
			expectedErr: true,
		},
		{
			description: "file exists",
			contents: map[string]string{
				"/path/to/foo": "",
			},
			path: "/path/to/foo",
		},
		{
			description: "symlink are followed, file exists",
			contents: map[string]string{
				"/x/foo":   "",
				"/path/to": "symlink=../x",
			},
			path: "/path/to/foo",
		},
		{
			description: "symlinks are followed, file does not exist",
			contents: map[string]string{
				"/x/bar":   "",
				"/path/to": "symlink=/x",
			},
			path:        "/path/to/foo",
			expectedErr: true,
		},
		{
			description: "symlinks are resolved within root",
			contents: map[string]string{
				"/x/foo":   "",
				"/path/to": "symlink=../../x",
			},
			path:     "/path/to/foo",
			skipGOOS: []string{"darwin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if slices.Contains(tc.skipGOOS, runtime.GOOS) {
				t.Skipf("test not supported on %s", runtime.GOOS)
			}

			containerRootDir := t.TempDir()
			for name, contents := range tc.contents {
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			root, _ := newRoot(containerRootDir)
			_, err := root.Open(tc.path)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCreate(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]string
		path        string
		expectedErr bool
		skipGOOS    []string
	}{
		{
			description: "empty path",
			path:        "",
			expectedErr: true,
		},
		{
			description: "file",
			path:        "foo",
			expectedErr: false,
		},
		{
			description: "parent dir does not exist",
			path:        "/path/to/foo",
			expectedErr: true,
		},
		{
			description: "parent dir exist",
			contents: map[string]string{
				"/path/to/bar": "",
			},
			path:        "/path/to/foo",
			expectedErr: false,
		},
		{
			description: "parent dir exist, relative symlinks followed",
			contents: map[string]string{
				"/x/y":     "",
				"/path/to": "symlink=../x",
			},
			path:        "/path/to/foo",
			expectedErr: false,
		},
		{
			description: "parent dir exist, absolute symlinks followed",
			contents: map[string]string{
				"/x/y":     "",
				"/path/to": "symlink=/x",
			},
			path:        "/path/to/foo",
			expectedErr: false,
			skipGOOS:    []string{"darwin"},
		},
		{
			description: "symlinks are resolved within root",
			contents: map[string]string{
				"/x/y":     "",
				"/path/to": "symlink=../../x",
			},
			path:        "/path/to/foo",
			expectedErr: false,
			skipGOOS:    []string{"darwin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if slices.Contains(tc.skipGOOS, runtime.GOOS) {
				t.Skipf("test not supported on %s", runtime.GOOS)
			}

			containerRootDir := t.TempDir()
			// (cdesiniotis) Iterate over the map's keys in a deterministic order.
			// This is necessary when preparing a parent directory prior to creating
			// files within that directory.
			for name, contents := range tc.contents {
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			root, _ := newRoot(containerRootDir)
			_, err := root.Create(tc.path)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			_, err = root.Open(tc.path)
			require.NoError(t, err)
		})
	}
}

func TestHasPath(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]string
		path        string
		expected    bool
		skipGOOS    []string
	}{
		{
			description: "path does not exist",
			path:        "/foo",
		},
		{
			description: "directory exists, absolute path",
			contents: map[string]string{
				"/foo/bar": "",
			},
			path:     "/foo",
			expected: true,
		},
		{
			description: "directory exists, path relative to root",
			contents: map[string]string{
				"/foo/bar": "",
			},
			path:     "foo",
			expected: true,
		},
		{
			description: "file exists, absolute path",
			contents: map[string]string{
				"/foo/bar": "",
			},
			path:     "/foo/bar",
			expected: true,
		},
		{
			description: "path is symlink, symlink is relative and within root",
			contents: map[string]string{
				"/x":       "",
				"/foo/bar": "symlink=../x",
			},
			path:     "/foo/bar",
			expected: true,
		},
		{
			description: "path is symlink, symlink is relative and not within root",
			contents: map[string]string{
				"/x":       "",
				"/foo/bar": "symlink=../../x",
			},
			path:     "/foo/bar",
			expected: true,
			skipGOOS: []string{"darwin"},
		},
		{
			description: "path is symlink, symlink is an absolute path",
			contents: map[string]string{
				"/x":       "",
				"/foo/bar": "symlink=/x",
			},
			path:     "/foo/bar",
			expected: true,
			skipGOOS: []string{"darwin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if slices.Contains(tc.skipGOOS, runtime.GOOS) {
				t.Skipf("test not supported on %s", runtime.GOOS)
			}

			containerRootDir := t.TempDir()
			for name, contents := range tc.contents {
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			root, _ := newRoot(containerRootDir)
			got := root.hasPath(tc.path)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestGlobFiles(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]string
		pattern     string
		expectedErr bool
		expected    []string
		skipGOOS    []string
	}{
		{
			description: "dir does not exist",
			pattern:     "/path/to/dir/libcuda.so.*.*",
			expectedErr: true,
		},
		{
			description: "pattern does not contain a directory",
			contents: map[string]string{
				"libcuda.so.1.1": "",
			},
			pattern:  "libcuda.so.*.*",
			expected: []string{"libcuda.so.1.1"},
		},
		{
			description: "no match",
			contents: map[string]string{
				"/path/to/dir/libfoo.so.1.1": "",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: nil,
		},
		{
			description: "one match",
			contents: map[string]string{
				"/path/to/dir/libcuda.so.1.1": "",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: []string{"/path/to/dir/libcuda.so.1.1"},
		},
		{
			description: "multiple matches",
			contents: map[string]string{
				"/path/to/dir/libcuda.so.1.1": "",
				"/path/to/dir/libcuda.so.1.2": "",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: []string{"/path/to/dir/libcuda.so.1.1", "/path/to/dir/libcuda.so.1.2"},
		},
		{
			description: "symlinks ignored",
			contents: map[string]string{
				"/path/to/dir/libcuda.so.1.1": "symlink=../foo",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: nil,
		},
		{
			description: "directories ignored",
			contents: map[string]string{
				"/path/to/dir/libcuda.so.1.1/foo": "",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: nil,
		},
		{
			description: "parent dir is symlink to path within root",
			contents: map[string]string{
				"/otherDir/foo":               "",
				"/path/to":                    "symlink=../otherDir",
				"/path/to/dir/libcuda.so.1.1": "",
			},
			pattern:  "/path/to/dir/libcuda.so.*.*",
			expected: []string{"/otherDir/dir/libcuda.so.1.1"},
			skipGOOS: []string{"darwin"},
		},
		{
			description: "parent dir is symlink to path not within root",
			contents: map[string]string{
				"/path/to":                    "symlink=../../",
				"/path/to/dir/libcuda.so.1.1": "",
			},
			pattern:     "/path/to/dir/libcuda.so.*.*",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if slices.Contains(tc.skipGOOS, runtime.GOOS) {
				t.Skipf("test not supported on %s", runtime.GOOS)
			}

			containerRootDir := t.TempDir()
			// (cdesiniotis) Iterate over the map's keys in a deterministic order.
			// This is necessary when preparing a parent directory prior to creating
			// files within that directory.
			for _, name := range slices.Sorted(maps.Keys(tc.contents)) {
				contents := tc.contents[name]
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			root, _ := newRoot(containerRootDir)
			got, err := root.globFiles(tc.pattern)

			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tc.expected, got)
		})
	}
}
