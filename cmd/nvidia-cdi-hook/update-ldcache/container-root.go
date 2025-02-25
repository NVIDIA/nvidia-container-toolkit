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

package ldcache

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

// resolve returns the absolute path including root path.
// Symlinks are resolved, but are guaranteed to resolve in the root.
func (r containerRoot) resolve(path string) (string, error) {
	absolute := filepath.Clean(filepath.Join(string(r), path))
	return symlink.FollowSymlinkInScope(absolute, string(r))
}
