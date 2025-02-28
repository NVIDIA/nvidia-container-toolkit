/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package utils

import (
	"os"
	"path/filepath"

	"github.com/moby/sys/symlink"
)

// A ContainerRoot represents the root filesystem of a container.
type ContainerRoot string

// GlobFiles matches the specified pattern in the container root.
// The files that match must be regular files.
func (r ContainerRoot) GlobFiles(pattern string) ([]string, error) {
	patternPath, err := r.Resolve(pattern)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(patternPath)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil {
			return nil, err
		}
		// Ignore symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Ignore directories.
		if info.IsDir() {
			continue
		}
		files = append(files, match)
	}
	return files, nil
}

// HasPath checks whether the specified path exists in the root.
func (r ContainerRoot) HasPath(path string) bool {
	resolved, err := r.Resolve(path)
	if err != nil {
		return false
	}
	if _, err := os.Stat(resolved); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// Resolve returns the absolute path including root path.
// Symlinks are resolved, but are guaranteed to resolve in the root.
func (r ContainerRoot) Resolve(path string) (string, error) {
	absolute := filepath.Clean(filepath.Join(string(r), path))
	return symlink.FollowSymlinkInScope(absolute, string(r))
}
