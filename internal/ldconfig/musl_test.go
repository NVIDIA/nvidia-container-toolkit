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
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateMuslPathFilesIfRequired(t *testing.T) {
	archs, ok := muslArchs[runtime.GOARCH]
	if !ok {
		t.Skip("musl .path files are not handled on this platform")
	}

	testCases := []struct {
		description string
		// isMusl adds the native musl dynamic linker.
		isMusl bool
		// hasCompat32Loader adds the 32-bit musl dynamic linker.
		hasCompat32Loader bool
		// pathFileContents is the contents of the native .path file before the update.
		pathFileContents *string
		driverDirs       []string
		systemDirs       []string
		expectedNative   *string
		expectedCompat32 *string
	}{
		{
			description: "glibc container is not modified",
			driverDirs:  []string{"/native"},
			systemDirs:  []string{"/lib", "/usr/lib"},
		},
		{
			description:    "path file is created with the default search path preserved",
			isMusl:         true,
			driverDirs:     []string{"/native"},
			systemDirs:     []string{"/lib", "/usr/lib"},
			expectedNative: ptr("/native\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "driver dirs are prepended to the existing contents",
			isMusl:           true,
			pathFileContents: ptr("/lib:/usr/local/lib:/usr/lib"),
			driverDirs:       []string{"/native"},
			systemDirs:       []string{"/lib", "/usr/lib"},
			expectedNative:   ptr("/native\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "dirs that are searched already are not reordered",
			isMusl:           true,
			pathFileContents: ptr("/lib\n/usr/local/lib\n/native\n"),
			driverDirs:       []string{"/native"},
			expectedNative:   ptr("/lib\n/usr/local/lib\n/native\n"),
		},
		{
			description:      "32-bit dirs are not added to the native path file",
			isMusl:           true,
			pathFileContents: ptr("/lib:/usr/local/lib:/usr/lib"),
			driverDirs:       []string{"/native", "/compat32"},
			systemDirs:       []string{"/lib", "/usr/lib"},
			expectedNative:   ptr("/native\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:       "32-bit dirs are added to the 32-bit path file if a loader is present",
			isMusl:            true,
			hasCompat32Loader: true,
			pathFileContents:  ptr("/lib:/usr/local/lib:/usr/lib"),
			driverDirs:        []string{"/native", "/compat32"},
			systemDirs:        []string{"/lib", "/usr/lib"},
			expectedNative:    ptr("/native\n/lib\n/usr/local/lib\n/usr/lib\n"),
			expectedCompat32:  ptr("/compat32\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:       "dirs with libraries of both classes are added to both path files",
			isMusl:            true,
			hasCompat32Loader: true,
			driverDirs:        []string{"/mixed"},
			expectedNative:    ptr("/mixed\n/lib\n/usr/local/lib\n/usr/lib\n"),
			expectedCompat32:  ptr("/mixed\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:    "dirs without libraries are added to the native path file",
			isMusl:         true,
			driverDirs:     []string{"/empty", "/not-a-lib", "/does-not-exist"},
			expectedNative: ptr("/empty\n/not-a-lib\n/does-not-exist\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:       "order is maintained for both classes",
			isMusl:            true,
			hasCompat32Loader: true,
			driverDirs:        []string{"/compat32", "/native", "/mixed"},
			expectedNative:    ptr("/native\n/mixed\n/lib\n/usr/local/lib\n/usr/lib\n"),
			expectedCompat32:  ptr("/compat32\n/mixed\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:      "entries separated by colons and newlines are read",
			isMusl:           true,
			pathFileContents: ptr("/lib:/usr/local/lib\n/usr/lib"),
			driverDirs:       []string{"/native"},
			expectedNative:   ptr("/native\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description:    "system dirs alone create the path file",
			isMusl:         true,
			systemDirs:     []string{"/lib", "/usr/lib"},
			expectedNative: ptr("/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description: "no dirs leave the container untouched",
			isMusl:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			root := t.TempDir()
			makeLibDir(t, root, "/native", elf.ELFCLASS64)
			makeLibDir(t, root, "/compat32", elf.ELFCLASS32)
			makeLibDir(t, root, "/mixed", elf.ELFCLASS64, elf.ELFCLASS32)
			makeLibDir(t, root, "/empty")
			notALib := makeLibDir(t, root, "/not-a-lib")
			require.NoError(t, os.WriteFile(filepath.Join(notALib, "libnotelf.so.1"), []byte("#!/bin/sh\n"), 0o600))

			require.NoError(t, os.MkdirAll(filepath.Join(root, "/etc"), 0o755))
			if tc.pathFileContents != nil {
				require.NoError(t, os.WriteFile(muslPathFile(root, archs.native), []byte(*tc.pathFileContents), 0o600))
			}
			if tc.isMusl {
				makeMuslLoader(t, root, archs.native, elf.ELFCLASS64)
			}
			if tc.hasCompat32Loader {
				makeMuslLoader(t, root, archs.compat32, elf.ELFCLASS32)
			}

			require.NoError(t, createMuslPathFilesIfRequired(root, tc.driverDirs, tc.systemDirs))

			requireFileContents(t, muslPathFile(root, archs.native), tc.expectedNative)
			requireFileContents(t, muslPathFile(root, archs.compat32), tc.expectedCompat32)
		})
	}
}

func TestIsMusl(t *testing.T) {
	archs, ok := muslArchs[runtime.GOARCH]
	if !ok {
		t.Skip("musl .path files are not handled on this platform")
	}

	t.Run("glibc container", func(t *testing.T) {
		require.False(t, isMusl(t.TempDir(), archs.native))
	})

	t.Run("musl loader", func(t *testing.T) {
		root := t.TempDir()
		makeMuslLoader(t, root, archs.native, elf.ELFCLASS64)
		require.True(t, isMusl(root, archs.native))
	})

	t.Run("alpine release file", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "/etc"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "/etc/alpine-release"), nil, 0o600))
		require.True(t, isMusl(root, archs.native))
	})
}

// makeLibDir creates a directory in the specified root holding a library of
// each of the specified ELF classes.
func makeLibDir(t *testing.T, root string, dir string, classes ...elf.Class) string {
	t.Helper()

	path := filepath.Join(root, dir)
	require.NoError(t, os.MkdirAll(path, 0o755))
	for _, class := range classes {
		name := "libnvidia-" + class.String() + ".so.999.88.77"
		require.NoError(t, os.WriteFile(filepath.Join(path, name), elfFile(t, class), 0o600))
	}
	return path
}

// makeMuslLoader creates the musl dynamic linker for the specified
// architecture in the specified root.
func makeMuslLoader(t *testing.T, root string, arch string, class elf.Class) {
	t.Helper()

	path := muslLoader(root, arch)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, elfFile(t, class), 0o600))
}

// elfFile returns a minimal ELF file of the specified class: a header without
// program or section headers.
func elfFile(t *testing.T, class elf.Class) []byte {
	t.Helper()

	var ident [elf.EI_NIDENT]byte
	copy(ident[:], elf.ELFMAG)
	ident[elf.EI_CLASS] = byte(class)
	ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	ident[elf.EI_VERSION] = byte(elf.EV_CURRENT)

	var header any = elf.Header64{Ident: ident, Version: uint32(elf.EV_CURRENT)}
	if class == elf.ELFCLASS32 {
		header = elf.Header32{Ident: ident, Version: uint32(elf.EV_CURRENT)}
	}
	var contents bytes.Buffer
	require.NoError(t, binary.Write(&contents, binary.LittleEndian, header))
	return contents.Bytes()
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
