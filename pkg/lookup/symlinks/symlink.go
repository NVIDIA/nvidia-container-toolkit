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

package symlinks

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resolve returns the link target of the specified filename or the filename if it is not a link.
func Resolve(filename string) (string, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return filename, fmt.Errorf("failed to get file info: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return filename, nil
	}

	return os.Readlink(filename)
}

// ResolveChain returns the resolved symlink chain starting with the specified
// link.
// The symlink chain for a regular file consists of only the regular file.
func ResolveChain(filename string) ([]string, error) {
	seen := make(map[string]bool)
	var chain []string

	for {
		target, err := Resolve(filename)
		if err != nil {
			return nil, err
		}
		if seen[filename] {
			return chain, nil
		}
		seen[filename] = true
		chain = append(chain, filename)
		if !filepath.IsAbs(target) {
			target, err = filepath.Abs(filepath.Join(filepath.Dir(filename), target))
			if err != nil {
				return nil, fmt.Errorf("failed to construct absolute path: %v", err)
			}
		}
		filename = target
	}
}

// ForceCreate creates a specified symlink.
// If a file (or empty directory) exists at the path it is removed.
func ForceCreate(target string, link string) error {
	_, err := os.Lstat(link)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	if !os.IsNotExist(err) {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}
	return os.Symlink(target, link)
}
