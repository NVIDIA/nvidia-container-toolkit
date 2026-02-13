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
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup/symlinks"
)

type ldcacheLocator struct {
	logger     logger.Interface
	root       string
	isOptional bool

	libraries []string
}

var _ Locator = (*ldcacheLocator)(nil)

// NewLdcacheLocator creates a locator that allows libraries to be found using
// the ldcache.
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

	var libraries []string
	_, libs64 := cache.List()
	for _, library := range libs64 {
		chain, err := symlinks.ResolveChain(library)
		if err != nil {
			b.logger.Warningf("Failed to resolve symlink chain for library %q: %v", library, err)
			continue
		}
		libraries = append(libraries, chain...)
	}

	l := &ldcacheLocator{
		logger:     b.logger,
		root:       b.root,
		isOptional: b.isOptional,
		libraries:  libraries,
	}

	return AsUnique(WithEvaluatedSymlinks(l))
}

// Locate finds the specified pattern in the libraries in the ldcache.
// If the pattern is a path (includes a slash), the locator root is prefixed to
// the pattern and libraries in the ldcache that match this pattern are
// returned. If the pattern is a filename, the pattern is compared to the
// basename of the libraries in the ldcache instead.
func (l *ldcacheLocator) Locate(pattern string) ([]string, error) {
	matcher := l.newPathPatternMatcher(pattern)

	var matches []string
	for _, library := range l.libraries {
		if !matcher.Match(library) {
			continue
		}
		matches = append(matches, library)
	}

	if len(matches) == 0 && !l.isOptional {
		return nil, fmt.Errorf("%s: %w", pattern, ErrNotFound)
	}

	return matches, nil
}

type fullMatcher string
type basenameMatcher string
type matcher interface {
	Match(string) bool
}

func (l *ldcacheLocator) newPathPatternMatcher(pattern string) matcher {
	if strings.Contains(pattern, "/") {
		return fullMatcher(filepath.Join(l.root, pattern))
	}
	return basenameMatcher(pattern)
}

func (m fullMatcher) Match(input string) bool {
	matches, err := filepath.Match(string(m), input)
	if err != nil {
		return false
	}
	return matches
}

func (m basenameMatcher) Match(input string) bool {
	return (fullMatcher)(m).Match(filepath.Base(input))
}
