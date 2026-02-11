/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup/symlinks"
)

type symlinkChain struct {
	logger  logger.Interface
	locator Locator
}

type symlink struct {
	locator Locator
}

// NewSymlinkChainLocator creats a locator that can be used for locating files through symlinks.
func NewSymlinkChainLocator(opts ...Option) Locator {
	f := NewFactory(opts...)
	l := &symlinkChain{
		logger:  f.logger,
		locator: f.NewFileLocator(),
	}

	return l
}

// NewSymlinkLocator creats a locator that can be used for locating files through symlinks.
func NewSymlinkLocator(opts ...Option) Locator {
	f := NewFactory(opts...)
	return AsUnique(WithEvaluatedSymlinks(f.NewFileLocator()))
}

// WithEvaluatedSymlinks wraps a locator in one that ensures that returned
// symlinks are resolved.
func WithEvaluatedSymlinks(locator Locator) Locator {
	l := symlink{
		locator: locator,
	}
	return &l
}

// Locate finds the specified pattern at the specified root.
// If the file is a symlink, the link is followed and all candidates to the final target are returned.
func (p symlinkChain) Locate(pattern string) ([]string, error) {
	candidates, err := p.locator.Locate(pattern)
	if err != nil {
		return nil, err
	}
	var filenames []string
	found := make(map[string]bool)

	for _, candidate := range candidates {
		if found[candidate] {
			continue
		}
		targets, err := symlinks.ResolveChain(candidate)
		if err != nil {
			return nil, fmt.Errorf("error resolving symlink chain: %w", err)
		}
		if len(targets) > 0 {
			p.logger.Debugf("Resolved link: %v", strings.Join(targets, " => "))
		}
		for _, target := range targets {
			if found[target] {
				continue
			}
			found[target] = true
			filenames = append(filenames, target)
		}
	}
	return filenames, nil
}

// Locate finds the specified pattern at the specified root.
// If the file is a symlink, the link is resolved and the target returned.
func (p symlink) Locate(pattern string) ([]string, error) {
	candidates, err := p.locator.Locate(pattern)
	if err != nil {
		return nil, err
	}

	var targets []string
	for _, candidate := range candidates {
		target, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve link: %w", err)
		}
		targets = append(targets, target)
	}
	return targets, err
}
