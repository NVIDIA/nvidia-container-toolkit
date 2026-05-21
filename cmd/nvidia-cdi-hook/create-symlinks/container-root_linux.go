//go:build linux

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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyphar/filepath-securejoin/pathrs-lite"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

func (m command) createSymlinkInRoot(containerRootDir string, targetPath string, linkPath string) error {
	rootDir, err := os.OpenFile(containerRootDir, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("failed to open container root: %w", err)
	}
	defer rootDir.Close()

	linkDir := filepath.Dir(linkPath)
	linkDirInRoot, err := pathrs.OpenatInRoot(rootDir, linkDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			linkDirInRoot, err = pathrs.MkdirAllHandle(rootDir, linkDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create parent dir of link in root: %w", err)
			}
		} else {
			return fmt.Errorf("failed to open parent dir of link in root: %w", err)
		}
	}
	linkDirFd := int(linkDirInRoot.Fd())

	m.logger.Infof("Symlinking %v to %v", filepath.Join(linkDirInRoot.Name(), filepath.Base(linkPath)), targetPath)

	tmpLink := filepath.Base(linkPath) + "-" + uuid.NewString()
	if err := unix.Symlinkat(targetPath, linkDirFd, tmpLink); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	if err := unix.Renameat(linkDirFd, tmpLink, linkDirFd, filepath.Base(linkPath)); err != nil {
		_ = unix.Unlinkat(linkDirFd, tmpLink, 0)
		return fmt.Errorf("failed to create symlink: %w", err)

	}
	return nil
}
