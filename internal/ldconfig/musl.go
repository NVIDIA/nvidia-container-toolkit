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
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// muslArchs maps the platform to the musl architecture names of the native
// dynamic linker and of the 32-bit dynamic linker the platform can also run.
// These name the linker, /lib/ld-musl-<arch>.so.1, and its .path file,
// /etc/ld-musl-<arch>.path.
var muslArchs = map[string]struct{ native, compat32 string }{
	"amd64": {"x86_64", "i386"},
	"arm64": {"aarch64", "armhf"},
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

// createMuslPathFilesIfRequired adds the specified directories to the musl
// .path files in the specified root, which musl searches instead of an ldcache.
//
// Unlike the ldcache, a .path file carries no architecture information: musl
// loads the first file matching the requested name and fails instead of
// searching on if that file is of another ELF class. Each directory is
// therefore added only to the .path file of the class of its libraries, and
// the driver directories are searched before the directories that may hold
// files of the same names for another class.
func createMuslPathFilesIfRequired(root string, driverDirs []string, systemDirs []string) error {
	archs, ok := muslArchs[runtime.GOARCH]
	if !ok || !isMusl(root, archs.native) {
		return nil
	}

	// Both supported platforms are 64-bit. Directories holding no libraries
	// are added to the native .path file.
	var nativeDirs, compat32Dirs []string
	for _, dir := range driverDirs {
		classes := libraryClasses(filepath.Join(root, dir))
		if len(classes) == 0 || classes[elf.ELFCLASS64] {
			nativeDirs = append(nativeDirs, dir)
		}
		if classes[elf.ELFCLASS32] {
			compat32Dirs = append(compat32Dirs, dir)
		}
	}

	if err := updateMuslPathFile(muslPathFile(root, archs.native), nativeDirs, systemDirs); err != nil {
		return err
	}
	// 32-bit libraries are only of use to a 32-bit dynamic linker.
	if len(compat32Dirs) == 0 || !isFile(muslLoader(root, archs.compat32)) {
		return nil
	}
	return updateMuslPathFile(muslPathFile(root, archs.compat32), compat32Dirs, nil)
}

// updateMuslPathFile writes the driver directories that are not searched
// already, then the existing entries, then the system directories to the
// specified .path file. The default search path stands in for a missing file.
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

// libraryClasses returns the ELF classes of the libraries in the specified
// directory. Files that are not ELF files are ignored.
func libraryClasses(dir string) map[elf.Class]bool {
	classes := make(map[elf.Class]bool)
	libraries, _ := filepath.Glob(filepath.Join(dir, "lib?*.so*"))
	for _, library := range libraries {
		f, err := elf.Open(library)
		if err != nil {
			continue
		}
		classes[f.Class] = true
		_ = f.Close()
	}
	return classes
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
