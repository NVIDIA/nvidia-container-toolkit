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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// muslArchs maps the platform to the musl architecture name, which names the
// dynamic linker, /lib/ld-musl-<arch>.so.1, and its .path file,
// /etc/ld-musl-<arch>.path.
var muslArchs = map[string]string{
	"amd64": "x86_64",
	"arm64": "aarch64",
}

// muslDefaultSearchPath is searched by musl when no .path file exists.
// Creating the file replaces it, so it is written out explicitly.
var muslDefaultSearchPath = []string{"/lib", "/usr/local/lib", "/usr/lib"}

func muslLoader(root, arch string) string {
	return filepath.Join(root, "/lib/ld-musl-"+arch+".so.1")
}

func muslPathFile(root, arch string) string {
	return filepath.Join(root, "/etc/ld-musl-"+arch+".path")
}

// createMuslPathFileIfRequired adds the specified directories to the musl
// .path file in the specified root, which musl searches instead of an ldcache.
func createMuslPathFileIfRequired(root string, driverDirs []string, systemDirs []string) error {
	arch, ok := muslArchs[runtime.GOARCH]
	if !ok || !isMusl(root, arch) {
		return nil
	}

	return updateMuslPathFile(muslPathFile(root, arch), driverDirs, systemDirs)
}

// updateMuslPathFile writes the driver directories that are not searched
// already, then the existing entries, then the system directories to the
// specified .path file. The default search path stands in for a missing file.
//
// The driver directories are searched first so that an injected library wins a
// name lookup against a file of the same name that happens to sit in a
// directory that is searched already. This is the precedence that the glibc
// path gives these directories through the 00-nvcr-*.conf drop-in.
func updateMuslPathFile(path string, driverDirs []string, systemDirs []string) error {
	if len(driverDirs) == 0 && len(systemDirs) == 0 {
		return nil
	}

	existing := muslDefaultSearchPath
	if contents, err := os.ReadFile(path); err == nil {
		// musl splits the file on colons and newlines.
		existing = strings.FieldsFunc(string(contents), func(r rune) bool { return r == ':' || r == '\n' })
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not read .path file: %w", err)
	}

	var dirs []string
	for _, dir := range driverDirs {
		if !slices.Contains(existing, dir) {
			dirs = append(dirs, dir)
		}
	}
	dirs = append(append(dirs, existing...), systemDirs...)

	pathFile, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open .path file: %w", err)
	}
	defer func() {
		_ = pathFile.Close()
	}()

	return outputListToFile(pathFile, dirs...)
}

// isMusl checks whether the container is running musl instead of glibc: its
// dynamic linker is present or, failing that, the container is Alpine-based.
func isMusl(root string, arch string) bool {
	return isFile(muslLoader(root, arch)) || isFile(filepath.Join(root, "/etc/alpine-release"))
}

// isFile checks whether the specified path exists and is not a directory.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
