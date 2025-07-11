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

package installer

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

// An artifactRoot is used as a source for installed artifacts.
// It is refined by a directory path, a library locator, and an executable locator.
type artifactRoot struct {
	path        string
	libraries   lookup.Locator
	executables lookup.Locator
}

func newArtifactRoot(logger logger.Interface, rootDirectoryPath string) (*artifactRoot, error) {
	relativeLibrarySearchPaths := []string{
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
	}
	var librarySearchPaths []string
	for _, l := range relativeLibrarySearchPaths {
		librarySearchPaths = append(librarySearchPaths, filepath.Join(rootDirectoryPath, l))
	}

	a := artifactRoot{
		path: rootDirectoryPath,
		libraries: lookup.NewLibraryLocator(
			lookup.WithLogger(logger),
			lookup.WithCount(1),
			lookup.WithSearchPaths(librarySearchPaths...),
		),
		executables: lookup.NewExecutableLocator(
			logger,
			rootDirectoryPath,
		),
	}

	return &a, nil
}

func (r *artifactRoot) findLibrary(name string) (string, error) {
	candidates, err := r.libraries.Locate(name)
	if err != nil {
		return "", fmt.Errorf("error locating library: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("library %v not found", name)
	}

	return candidates[0], nil
}

func (r *artifactRoot) findExecutable(name string) (string, error) {
	candidates, err := r.executables.Locate(name)
	if err != nil {
		return "", fmt.Errorf("error locating executable: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("executable %v not found", name)
	}

	return candidates[0], nil
}
