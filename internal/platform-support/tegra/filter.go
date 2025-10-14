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
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

type ignoreSymlinkMountSpecPatterns []string

func (d ignoreSymlinkMountSpecPatterns) match(name string) bool {
	for _, pattern := range d {
		target := name
		if strings.HasPrefix(pattern, "**/") {
			target = filepath.Base(name)
			pattern = strings.TrimPrefix(pattern, "**/")
		}
		if match, _ := filepath.Match(pattern, target); match {
			return true
		}
	}
	return false
}

func (d ignoreSymlinkMountSpecPatterns) filter(input ...string) []string {
	var filtered []string
	for _, name := range input {
		if d.match(name) {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

func (d ignoreSymlinkMountSpecPatterns) Apply(input MountSpecPathsByTyper) MountSpecPathsByTyper {
	ms := input.MountSpecPathsByType()

	if symlinks, ok := ms[csv.MountSpecSym]; ok {
		ms[csv.MountSpecSym] = d.filter(symlinks...)
	}

	return ms
}

// A filter removes elements from an input list and returns the remaining
// elements.
type filter interface {
	apply(...string) []string
}

// A stringMatcher implements the MatchString function.
type stringMatcher interface {
	MatchString(string) bool
}

// A matcherAsFilter is used to ensure that a string matcher can be used as a filter.
type matcherAsFilter struct {
	stringMatcher
}

type filterByMountSpecType map[csv.MountSpecType]filter

type pathPatterns []string
type pathPattern string
type basenamePattern string

// MatchString for a set of path patterns returns true if any of the patterns
// matches against the input string.
func (d pathPatterns) MatchString(input string) bool {
	for _, pattern := range d {
		if match := pathPattern(pattern).MatchString(input); match {
			return true
		}
	}
	return false
}

// MatchString attempts to match a path pattern to the specified input string.
// If the pattern starts with `**/` the input is treated as a path and only
// the basenames are matched using regular glob rules.
func (d pathPattern) MatchString(input string) bool {
	if strings.HasPrefix(string(d), "**/") {
		return basenamePattern(d).MatchString(input)
	}
	match, _ := filepath.Match(string(d), input)
	return match
}

// MatchString for a basename pattern applies the specified pattern against the
// basename of the input.
// If the pattern starts with **/, this is stripped before attempting to match.
func (d basenamePattern) MatchString(input string) bool {
	pattern := strings.TrimPrefix(string(d), "**/")
	match, _ := filepath.Match(pattern, filepath.Base(input))
	return match
}

// Apply the specified per-type filters to the input mount specs.
func (p filterByMountSpecType) Apply(input MountSpecPathsByTyper) MountSpecPathsByTyper {
	ms := input.MountSpecPathsByType()
	for t, filter := range p {
		if len(ms[t]) == 0 {
			continue
		}
		ms[t] = filter.apply(ms[t]...)
	}
	return ms
}

// apply uses a matcher to filter an input string.
// Each element in the input that matches is skipped and the remaining elements
// are returned.
func (f *matcherAsFilter) apply(input ...string) []string {
	var filtered []string
	for _, path := range input {
		if f.MatchString(path) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

// removeAll is a filter that will not return any inputs.
type removeAll struct{}

func (a removeAll) apply(...string) []string {
	return nil
}
