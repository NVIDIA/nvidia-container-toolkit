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
	"path/filepath"
	"runtime"
)

// compat32Loaders maps the platform to the paths of the dynamic linkers that a
// container uses to run 32-bit applications.
var compat32Loaders = map[string][]string{
	"amd64": {"/lib/ld-linux.so.2", "/lib32/ld-linux.so.2"},
	"arm64": {"/lib/ld-linux-armhf.so.3", "/lib/ld-linux.so.3"},
}

// allowsCompat32 checks whether the 32-bit driver libraries are of use in the
// specified root.
//
// A container that uses musl never sees these. A musl .path file carries no
// architecture information and musl has no notion of a multiarch layout: it
// loads the first file matching the requested name and fails instead of
// searching on if that file is of another ELF class.
//
// Other containers see the 32-bit libraries if they requested the compat32
// driver capability -- as is the case for the nvidia-container-cli -- or if
// they ship a dynamic linker for 32-bit applications.
func allowsCompat32(root string, requested bool) bool {
	if arch, ok := muslArchs[runtime.GOARCH]; ok && isMusl(root, arch) {
		return false
	}
	if requested {
		return true
	}
	for _, loader := range compat32Loaders[runtime.GOARCH] {
		if isFile(filepath.Join(root, loader)) {
			return true
		}
	}
	return false
}

// excludeCompat32Directories returns the specified directories with those that
// hold 32-bit libraries removed. The directories are resolved relative to the
// specified root.
func excludeCompat32Directories(root string, dirs []string) []string {
	var filtered []string
	for _, dir := range dirs {
		if isCompat32Dir(filepath.Join(root, dir)) {
			continue
		}
		filtered = append(filtered, dir)
	}
	return filtered
}

// isCompat32Dir checks whether the specified directory holds 32-bit libraries
// and no 64-bit ones. Files that are not ELF files are ignored, as are
// directories that hold no libraries at all.
// Note that both supported platforms are 64-bit.
func isCompat32Dir(dir string) bool {
	var compat32 bool
	libraries, _ := filepath.Glob(filepath.Join(dir, "lib?*.so*"))
	for _, library := range libraries {
		f, err := elf.Open(library)
		if err != nil {
			continue
		}
		class := f.Class
		_ = f.Close()
		if class == elf.ELFCLASS64 {
			return false
		}
		compat32 = compat32 || class == elf.ELFCLASS32
	}
	return compat32
}
