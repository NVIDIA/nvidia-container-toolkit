/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package oci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/sys/symlink"
)

// A ContainerRoot represents the root directory of a container's filesystem.
type ContainerRoot string

// CreateLdsoconfdFile creates a file at /etc/ld.so.conf.d/ in the specified container root.
// The file is created at /etc/ld.so.conf.d/{{ .pattern }} using `CreateTemp` and
// contains the specified directories on each line.
func (r ContainerRoot) CreateLdsoconfdFile(pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		return nil
	}

	ldsoconfdDir, err := r.Resolve("/etc/ld.so.conf.d")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err = configFile.WriteString(fmt.Sprintf("%s\n", dir))
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}

// GlobFiles matches the specified pattern in the container root.
// The files that match must be regular files. Symlinks and directories are ignored.
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

// ToContainerPath converts the specified path to a path in the container.
// Relative paths and absolute paths that are in the container root are returned as is.
func (r ContainerRoot) ToContainerPath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	if !strings.HasPrefix(path, string(r)) {
		return path
	}

	return strings.TrimPrefix(path, string(r))
}
