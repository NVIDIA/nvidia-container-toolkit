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
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// collectLibraries locates and installs the libraries that are part of
// the nvidia-container-toolkit.
// A predefined set of library candidates are considered, with the first one
// resulting in success being installed to the toolkit folder. The install process
// resolves the symlink for the library and copies the versioned library itself.
func (t *toolkitInstaller) collectLibraries() ([]Installer, error) {
	requiredLibraries := []string{
		"libnvidia-container.so.1",
		"libnvidia-container-go.so.1",
	}

	var installers []Installer
	for _, l := range requiredLibraries {
		libraryPath, err := t.artifactRoot.findLibrary(l)
		if err != nil {
			if t.ignoreErrors {
				log.Errorf("Ignoring error: %v", err)
				continue
			}
			return nil, err
		}

		installers = append(installers, library(libraryPath))

		if filepath.Base(libraryPath) == l {
			continue
		}

		link := symlink{
			linkname: l,
			target:   filepath.Base(libraryPath),
		}
		installers = append(installers, link)
	}

	return installers, nil
}

type library string

// Install copies the library l to the destination folder.
// The same basename is used in the destination folder.
func (l library) Install(destinationDir string) error {
	dest := filepath.Join(destinationDir, filepath.Base(string(l)))

	_, err := installFile(string(l), dest)
	return err
}
