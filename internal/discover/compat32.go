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

package discover

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

// Compat32Directory is where a container that asks for them gets the 32-bit
// driver libraries. It goes into the ldcache like any other folder, where
// glibc's dynamic linker uses it for 32-bit programs alone, but never into
// musl's .path file.
const Compat32Directory = "/usr/lib/nvidia/compat32"

// NewCompat32Discoverer discovers the 32-bit libraries of the driver: those
// with its version suffix that the 32-bit entries of the driver root's ldcache
// list. They are mounted into Compat32Directory, which the ldcache update adds,
// rather than at their host paths: a spec cannot know the container's layout,
// and a host's 32-bit directory can be a 64-bit one in the container, as a RHEL
// host's /usr/lib is in an Arch container, where /usr/lib64 links to /usr/lib.
func NewCompat32Discoverer(logger logger.Interface, driver *root.Driver, hookCreator HookCreator) (Discover, error) {
	version, err := driver.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to get driver version: %w", err)
	}

	locator := lookup.NewLibrary32Locator(
		lookup.WithLogger(logger),
		lookup.WithRoot(driver.Root),
	)
	return newCompat32Discoverer(logger, locator, version, hookCreator), nil
}

func newCompat32Discoverer(logger logger.Interface, locator lookup.Locator, version string, hookCreator HookCreator) Discover {
	libraries := &compat32Libraries{
		logger:  logger,
		locator: locator,
		version: version,
	}
	ldcacheUpdate, _ := NewLDCacheUpdateHook(logger, libraries, hookCreator)
	return Merge(libraries, ldcacheUpdate)
}

type compat32Libraries struct {
	None
	logger  logger.Interface
	locator lookup.Locator
	version string
}

// Mounts returns a read-only mount in Compat32Directory for each 32-bit
// driver library; none where the driver has no 32-bit libraries installed.
func (d *compat32Libraries) Mounts() ([]Mount, error) {
	libraries, err := d.locator.Locate("*.so." + d.version)
	if errors.Is(err, lookup.ErrNotFound) {
		d.logger.Infof("No 32-bit libraries found for driver version %v", d.version)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to locate 32-bit libraries for driver version %v: %w", d.version, err)
	}

	var mounts []Mount
	for _, library := range libraries {
		mounts = append(mounts, Mount{
			HostPath: library,
			Path:     filepath.Join(Compat32Directory, filepath.Base(library)),
			Options: []string{
				"ro",
				"nosuid",
				"nodev",
				"rbind",
				"rprivate",
			},
		})
	}
	return mounts, nil
}
