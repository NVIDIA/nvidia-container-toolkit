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

package tegra

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup/symlinks"
)

// New creates a new tegra discoverer using the supplied functional options.
func New(opts ...Option) (discover.Discover, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.devRoot == "" {
		o.devRoot = o.driverRoot
	}

	if o.symlinkLocator == nil {
		o.symlinkLocator = lookup.NewSymlinkLocator(
			lookup.WithLogger(o.logger),
			lookup.WithRoot(o.driverRoot),
			lookup.WithSearchPaths(append(o.librarySearchPaths, "/")...),
		)
	}

	if o.symlinkChainLocator == nil {
		o.symlinkChainLocator = lookup.NewSymlinkChainLocator(
			lookup.WithLogger(o.logger),
			lookup.WithRoot(o.driverRoot),
		)
	}

	if o.resolveSymlink == nil {
		o.resolveSymlink = symlinks.Resolve
	}

	mountSpecDiscoverer := o.newDiscovererFromMountSpecs(o.mountSpecs.MountSpecPathsByType())

	tegraSystemMounts := discover.NewMounts(
		o.logger,
		lookup.NewFileLocator(lookup.WithLogger(o.logger)),
		"",
		[]string{
			"/etc/nv_tegra_release",
		},
	)

	d := discover.Merge(
		mountSpecDiscoverer,
		tegraSystemMounts,
	)

	return d, nil
}
