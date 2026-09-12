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

func TestAllowsCompat32(t *testing.T) {
	loaders, ok := compat32Loaders[runtime.GOARCH]
	if !ok {
		t.Skip("32-bit libraries are not handled on this platform")
	}
	muslArch := muslArchs[runtime.GOARCH]

	testCases := []struct {
		description string
		// compat32Loader adds a dynamic linker for 32-bit applications.
		compat32Loader string
		// isMusl adds the musl dynamic linker.
		isMusl bool
		// requested indicates that the compat32 driver capability was requested.
		requested bool
		expected  bool
	}{
		{
			description: "64-bit-only container is skipped",
		},
		{
			description:    "container with a 32-bit loader is allowed",
			compat32Loader: loaders[0],
			expected:       true,
		},
		{
			description: "container that requested the capability is allowed",
			requested:   true,
			expected:    true,
		},
		{
			description: "musl container is skipped",
			isMusl:      true,
		},
		{
			description: "musl container that requested the capability is skipped",
			isMusl:      true,
			requested:   true,
		},
		{
			description:    "musl container with a 32-bit loader is skipped",
			isMusl:         true,
			compat32Loader: loaders[0],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			root := t.TempDir()
			if tc.compat32Loader != "" {
				makeFile(t, filepath.Join(root, tc.compat32Loader))
			}
			if tc.isMusl {
				makeMuslLoader(t, root, muslArch)
			}

			require.Equal(t, tc.expected, allowsCompat32(root, tc.requested))
		})
	}
}

func TestAllowsCompat32DetectsAllLoaders(t *testing.T) {
	for _, loader := range compat32Loaders[runtime.GOARCH] {
		t.Run(loader, func(t *testing.T) {
			root := t.TempDir()
			makeFile(t, filepath.Join(root, loader))

			require.True(t, allowsCompat32(root, false))
		})
	}
}

func TestExcludeCompat32Directories(t *testing.T) {
	root := t.TempDir()
	makeLibDir(t, root, "/native", elf.ELFCLASS64)
	makeLibDir(t, root, "/compat32", elf.ELFCLASS32)
	makeLibDir(t, root, "/mixed", elf.ELFCLASS64, elf.ELFCLASS32)
	makeLibDir(t, root, "/empty")
	notALib := makeLibDir(t, root, "/not-a-lib")
	require.NoError(t, os.WriteFile(filepath.Join(notALib, "libnotelf.so.1"), []byte("#!/bin/sh\n"), 0o600))

	testCases := []struct {
		description string
		dirs        []string
		expected    []string
	}{
		{
			description: "32-bit dirs are removed",
			dirs:        []string{"/native", "/compat32"},
			expected:    []string{"/native"},
		},
		{
			description: "dirs with libraries of both classes are kept",
			dirs:        []string{"/mixed"},
			expected:    []string{"/mixed"},
		},
		{
			description: "dirs without libraries are kept",
			dirs:        []string{"/empty", "/not-a-lib", "/does-not-exist"},
			expected:    []string{"/empty", "/not-a-lib", "/does-not-exist"},
		},
		{
			description: "the order of the remaining dirs is maintained",
			dirs:        []string{"/compat32", "/mixed", "/native"},
			expected:    []string{"/mixed", "/native"},
		},
		{
			description: "32-bit dirs alone leave no dirs",
			dirs:        []string{"/compat32"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			require.Equal(t, tc.expected, excludeCompat32Directories(root, tc.dirs))
		})
	}
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

// makeFile creates an empty file at the specified path.
func makeFile(t *testing.T, path string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, nil, 0o600))
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
