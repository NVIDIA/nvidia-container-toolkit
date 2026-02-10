/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package lookup

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/go-nvml/pkg/dl"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// dlopenLocator can be used to locate libraries given a system's dynamic
// linker.
type dlopenLocator struct {
	logger logger.Interface
}

// newDlopenLocator creates a locator that can be used for locating libraries
// through the dlopen mechanism.
func (f *Factory) newDlopenLocator() Locator {
	d := &dlopenLocator{
		logger: f.logger,
	}

	return d
}

// Locate finds the specified pattern if the systems' dynamic linker can find
// it via dlopen. Note that patterns with wildcard patterns will likely not be
// found as it is uncommon for libraries to have wildcard patterns in their
// file name.
func (d dlopenLocator) Locate(pattern string) ([]string, error) {
	// Create a new library using the `RTLD_LOCAL` flag since we do not want to
	// add loaded symbols for resolution here.
	lib := dl.New(pattern, dl.RTLD_LAZY|dl.RTLD_LOCAL)
	if err := lib.Open(); err != nil {
		return nil, fmt.Errorf("failed to load library %v: %w", pattern, err)
	}
	defer func() {
		_ = lib.Close()
	}()

	libPath, err := lib.Path()
	if err != nil {
		return nil, fmt.Errorf("failed to get library path: %w", err)
	}
	d.logger.Debugf("Found library for %s at %s", pattern, libPath)

	resolvedPath, err := filepath.EvalSymlinks(libPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve library symlink: %w", err)
	}
	if libPath != resolvedPath {
		d.logger.Debugf("Resolved library symlink %v => %v", libPath, resolvedPath)
	}

	return []string{resolvedPath}, nil
}
