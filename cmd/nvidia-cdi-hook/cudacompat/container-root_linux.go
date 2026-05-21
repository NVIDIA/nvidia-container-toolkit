//go:build linux

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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyphar/filepath-securejoin/pathrs-lite"
	"golang.org/x/sys/unix"
)

type root struct {
	*os.File
}

func newRoot(path string) (*root, error) {
	rootDir, err := os.OpenFile(path, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return &root{rootDir}, nil
}

// MkdirAll creates a new directory in the root, along with any necessary parents
func (r root) MkdirAll(path string, mode os.FileMode) error {
	d, err := pathrs.MkdirAllHandle(r.File, path, mode)
	if err != nil {
		return err
	}
	_ = d.Close()
	return nil
}

// Create creates the file in the root
func (r root) Create(path string) (*os.File, error) {
	dir, err := pathrs.OpenatInRoot(r.File, filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("failed to open directory in root: %w", err)
	}
	defer dir.Close()
	fd, err := unix.Openat(
		int(dir.Fd()),
		filepath.Base(path),
		unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %q in root: %w", path, err)
	}
	return os.NewFile(uintptr(fd), filepath.Join(r.Name(), path)), nil
}

// Open opens the named file in the root for reading
func (r root) Open(path string) (*os.File, error) {
	path = filepath.Clean(path)
	f, err := pathrs.OpenatInRoot(r.File, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file in root: %w", err)
	}
	defer f.Close()

	reopenedHandle, err := pathrs.Reopen(f, unix.O_RDONLY)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen file as read-only: %w", err)
	}

	return reopenedHandle, nil
}

// hasPath checks whether the specified path exists in the root.
func (r root) hasPath(path string) bool {
	path = filepath.Clean(path)
	f, err := pathrs.OpenatInRoot(r.File, path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// globFiles matches the specified pattern in the root.
// The files that match must be regular files.
func (r root) globFiles(pattern string) ([]string, error) {
	dir := filepath.Dir(pattern)
	basePattern := filepath.Base(pattern)

	d, err := pathrs.OpenatInRoot(r.File, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to open dir in root: %w", err)
	}
	defer d.Close()

	dirInRoot, err := pathrs.Reopen(d, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen directory as read-only: %w", err)
	}
	defer dirInRoot.Close()

	dEntries, err := dirInRoot.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of directory %q: %w", dir, err)
	}

	var files []string
	dir = strings.TrimPrefix(dirInRoot.Name(), r.Name())
	for _, dEntry := range dEntries {
		if match, err := filepath.Match(basePattern, dEntry.Name()); err != nil || !match {
			continue
		}
		if dType := dEntry.Type(); !dType.IsRegular() || dType.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, dEntry.Name()))
	}
	return files, nil
}
