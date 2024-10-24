/**
# Copyright 2024 NVIDIA CORPORATION
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
	"slices"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
)

type ldcacheLocator struct {
	*builder
	resolvesTo map[string]string
}

var _ Locator = (*ldcacheLocator)(nil)

func NewLdcacheLocator(opts ...Option) Locator {
	b := newBuilder(opts...)

	cache, err := ldcache.New(b.logger, b.root)
	if err != nil {
		b.logger.Warningf("Failed to load ldcache: %v", err)
		if b.isOptional {
			return &null{}
		}
		return &notFound{}
	}

	chain := NewSymlinkChainLocator(WithOptional(true))

	resolvesTo := make(map[string]string)
	_, libs64 := cache.List()
	for _, library := range libs64 {
		if _, processed := resolvesTo[library]; processed {
			continue
		}
		candidates, err := chain.Locate(library)
		if err != nil {
			b.logger.Errorf("error processing library %s from ldcache: %v", library, err)
			continue
		}

		if len(candidates) == 0 {
			resolvesTo[library] = library
			continue
		}

		// candidates represents a symlink chain.
		// The first element represents the start of the chain and the last
		// element the final target.
		target := candidates[len(candidates)-1]
		for _, candidate := range candidates {
			resolvesTo[candidate] = target
		}
	}

	return &ldcacheLocator{
		builder:    b,
		resolvesTo: resolvesTo,
	}
}

// Locate finds the specified libraryname.
// If the input is a library name, the ldcache is searched otherwise the
// provided path is resolved as a symlink.
func (l ldcacheLocator) Locate(libname string) ([]string, error) {
	var matcher func(string, string) bool

	if filepath.IsAbs(libname) {
		matcher = func(p string, c string) bool {
			m, _ := filepath.Match(filepath.Join(l.root, p), c)
			return m
		}
	} else {
		matcher = func(p string, c string) bool {
			m, _ := filepath.Match(p, filepath.Base(c))
			return m
		}
	}

	var matches []string
	seen := make(map[string]bool)
	for name, target := range l.resolvesTo {
		if !matcher(libname, name) {
			continue
		}
		if seen[target] {
			continue
		}
		seen[target] = true
		matches = append(matches, target)
	}

	slices.Sort(matches)

	if len(matches) == 0 && !l.isOptional {
		return nil, fmt.Errorf("%s: %w", libname, ErrNotFound)
	}

	return matches, nil
}
