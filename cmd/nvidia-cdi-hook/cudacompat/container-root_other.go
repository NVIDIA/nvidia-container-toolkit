//go:build !linux

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

package cudacompat

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type root struct {
	*os.Root
}

func newRoot(path string) (*root, error) {
	r, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &root{r}, nil
}

// MkdirAll creates a new directory in the root, along with any necessary parents
func (r root) MkdirAll(path string, mode os.FileMode) error {
	return r.Root.MkdirAll(r.normalizePath(path), mode)
}

// Create creates the file in the root
func (r root) Create(path string) (*os.File, error) {
	return r.Root.Create(r.normalizePath(path))
}

func (r root) Open(path string) (*os.File, error) {
	return r.Root.Open(r.normalizePath(path))
}

// hasPath checks whether the specified path exists in the root.
func (r root) hasPath(path string) bool {
	f, err := r.Open(r.normalizePath(path))
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// globFiles matches the specified pattern in the root.
// The files that match must be regular files.
func (r root) globFiles(pattern string) ([]string, error) {
	pattern = filepath.Clean(pattern)
	dir := filepath.Dir(pattern)
	basePattern := filepath.Base(pattern)

	d, err := r.OpenRoot(r.normalizePath(dir))
	if err != nil {
		return nil, err
	}
	defer d.Close()

	matches, err := fs.Glob(d.FS(), basePattern)
	if err != nil {
		return nil, err
	}

	var files []string
	dir = r.normalizePath(strings.TrimPrefix(d.Name(), r.Name()))
	if dir != "" && dir != "." {
		dir = filepath.Join("/", dir)
	}
	for _, match := range matches {
		info, err := d.Lstat(match)
		if err != nil {
			return nil, err
		}
		// Ignore symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Ignore directories
		if info.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, match))
	}
	return files, nil
}

// normalizePath ensures that the specified path is always relative to the root.
func (r root) normalizePath(path string) string {
	return strings.TrimPrefix(filepath.Clean(path), "/")
}
