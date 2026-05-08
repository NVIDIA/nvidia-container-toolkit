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

package symlinks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/sys/symlink"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup/symlinks"
)

func (m command) createSymlinkInRoot(containerRootDir string, targetPath string, link string) error {
	linkPath := filepath.Join(containerRootDir, link)
	// We resolve the parent of the symlink that we're creating in the container root.
	// If we resolve the full link path, an existing link at the location itself
	// is also resolved here and we are unable to force create the link.
	resolvedLinkParent, err := symlink.FollowSymlinkInScope(filepath.Dir(linkPath), containerRootDir)
	if err != nil {
		return fmt.Errorf("failed to follow path for link %s relative to %s: %w", link, containerRootDir, err)
	}
	resolvedLinkPath := filepath.Join(resolvedLinkParent, filepath.Base(linkPath))

	m.logger.Infof("Symlinking %q to %q", resolvedLinkPath, targetPath)
	err = os.MkdirAll(filepath.Dir(resolvedLinkPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	err = symlinks.ForceCreate(targetPath, resolvedLinkPath)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}
