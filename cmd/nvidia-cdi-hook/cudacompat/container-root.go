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

package cudacompat

import (
	"os"
	"path/filepath"

	"github.com/moby/sys/symlink"
)

// A containerRoot represents the root filesystem of a container.
type containerRoot string

// hasPath checks whether the specified path exists in the root.
func (r containerRoot) hasPath(path string) bool {
	resolved, err := r.resolve(path)
	if err != nil {
		return false
	}
	if _, err := os.Stat(resolved); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// globFiles matches the specified pattern in the root.
// The files that match must be regular files.
func (r containerRoot) globFiles(pattern string) ([]string, error) {
	patternPath, err := r.resolve(pattern)
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

// resolve returns the absolute path including root path.
// Symlinks are resolved, but are guaranteed to resolve in the root.
func (r containerRoot) resolve(path string) (string, error) {
	absolute := filepath.Clean(filepath.Join(string(r), path))
	return symlink.FollowSymlinkInScope(absolute, string(r))
}
