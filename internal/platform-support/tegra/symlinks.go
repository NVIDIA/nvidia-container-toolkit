/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

type symlinkHook struct {
	discover.None
	logger            logger.Interface
	nvidiaCDIHookPath string
	targets           []string

	// The following can be overridden for testing
	symlinkChainLocator lookup.Locator
	resolveSymlink      func(string) (string, error)
}

// createCSVSymlinkHooks creates a discoverer for a hook that creates required symlinks in the container
func (o tegraOptions) createCSVSymlinkHooks(targets []string) discover.Discover {
	return symlinkHook{
		logger:              o.logger,
		nvidiaCDIHookPath:   o.nvidiaCDIHookPath,
		targets:             targets,
		symlinkChainLocator: o.symlinkChainLocator,
		resolveSymlink:      o.resolveSymlink,
	}
}

// Hooks returns a hook to create the symlinks from the required CSV files
func (d symlinkHook) Hooks() ([]discover.Hook, error) {
	return discover.CreateCreateSymlinkHook(
		d.nvidiaCDIHookPath,
		d.getCSVFileSymlinks(),
	).Hooks()
}

// getSymlinkCandidates returns a list of symlinks that are candidates for being created.
func (d symlinkHook) getSymlinkCandidates() []string {
	var candidates []string
	for _, target := range d.targets {
		reslovedSymlinkChain, err := d.symlinkChainLocator.Locate(target)
		if err != nil {
			d.logger.Warningf("Failed to locate symlink %v", target)
			continue
		}
		candidates = append(candidates, reslovedSymlinkChain...)
	}
	return candidates
}

func (d symlinkHook) getCSVFileSymlinks() []string {
	var links []string
	created := make(map[string]bool)
	// candidates is a list of absolute paths to symlinks in a chain, or the final target of the chain.
	for _, candidate := range d.getSymlinkCandidates() {
		target, err := d.resolveSymlink(candidate)
		if err != nil {
			d.logger.Debugf("Skipping invalid link: %v", err)
			continue
		} else if target == candidate {
			d.logger.Debugf("%v is not a symlink", candidate)
			continue
		}

		link := fmt.Sprintf("%v::%v", target, candidate)
		if created[link] {
			d.logger.Debugf("skipping duplicate link: %v", link)
			continue
		}
		created[link] = true

		links = append(links, link)
	}

	return links
}
