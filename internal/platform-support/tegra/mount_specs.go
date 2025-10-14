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

package tegra

import (
	"regexp"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

// A MountSpecPathsByTyper provides a function to return mount specs paths by
// mount type.
// The MountSpecTypes are one of: dev, dir, lib, sym and define how these should
// be included in a container (or represented in the associated CDI spec).
type MountSpecPathsByTyper interface {
	MountSpecPathsByType() MountSpecPathsByType
}

type MountSpecPathsByType map[csv.MountSpecType][]string

var _ MountSpecPathsByTyper = (MountSpecPathsByType)(nil)

// MountSpecPathsByType for a variable of type MountSpecPathsByType returns the
// underlying data structure.
// This allows for using this type in functions such as Merge and Filter.
func (m MountSpecPathsByType) MountSpecPathsByType() MountSpecPathsByType {
	return m
}

type merge []MountSpecPathsByTyper

// Merge combines the MountSpecPathsByType for the specified sources.
func Merge(sources ...MountSpecPathsByTyper) MountSpecPathsByTyper {
	return merge(sources)
}

// MountSpecPathsByType for a set of merged mount specs combines the list of
// paths per type.
func (ts merge) MountSpecPathsByType() MountSpecPathsByType {
	targetsByType := make(MountSpecPathsByType)
	for _, t := range ts {
		if t == nil {
			continue
		}
		for tType, targets := range t.MountSpecPathsByType() {
			targetsByType[tType] = append(targetsByType[tType], targets...)
		}
	}
	return targetsByType
}

type Transformer interface {
	Apply(MountSpecPathsByTyper) MountSpecPathsByTyper
}

type transformMountSpecByPathsByType struct {
	Transformer
	input MountSpecPathsByTyper
}

// Transform applies the specified transform to a set of mount specs by paths.
func Transform(t Transformer, input MountSpecPathsByTyper) MountSpecPathsByTyper {
	return transformMountSpecByPathsByType{
		Transformer: t,
		input:       input,
	}
}

func (m transformMountSpecByPathsByType) MountSpecPathsByType() MountSpecPathsByType {
	return m.Apply(m.input).MountSpecPathsByType()
}

// AsIgnorePatternsByType uses the paths in the specified mount spec paths by
// mount spec type as patterns to ignore.
func AsIgnorePatternsByType(m MountSpecPathsByTyper) Transformer {
	patternsByType := m.MountSpecPathsByType()

	ignorePatterns := make(filterByMountSpecType)
	for t, patterns := range patternsByType {
		ignorePatterns[t] = &matcherAsFilter{pathPatterns(patterns)}
	}
	return ignorePatterns
}

// OnlyDeviceNodes creates a transformer that will remove any input mounts specs
// that are not of the `MountSpecDev` type.
func OnlyDeviceNodes() Transformer {
	return filterByMountSpecType{
		csv.MountSpecDir: removeAll{},
		csv.MountSpecLib: removeAll{},
		csv.MountSpecSym: removeAll{},
	}
}

// WithoutDeviceNodes creates a transformer that will remove entries with type
// MountSpecDevice from the input.
func WithoutDeviceNodes() Transformer {
	return filterByMountSpecType{
		csv.MountSpecDev: removeAll{},
	}
}

// WithoutRegularDeviceNodes creates a transfomer which removes
// regular `/dev/nvidia[0-9]+` device nodes from the source.
func WithoutRegularDeviceNodes() Transformer {
	return filterByMountSpecType{
		csv.MountSpecDev: &matcherAsFilter{regexp.MustCompile("^/dev/nvidia[0-9]+$")},
	}
}

// DeviceNodes creates a set of MountSpecPaths for the specified device nodes.
// These have the MoutSpecDev type.
func DeviceNodes(dn ...string) MountSpecPathsByTyper {
	return MountSpecPathsByType{
		csv.MountSpecDev: dn,
	}
}

// DeviceNodes creates a set of MountSpecPaths for the specified symlinks.
// These have the MountSpecSym type.
func Symlinks(s ...string) MountSpecPathsByTyper {
	return MountSpecPathsByType{
		csv.MountSpecSym: s,
	}
}
