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

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

// builder defines the builder for a file locator.
type builder struct {
	logger      logger.Interface
	root        string
	searchPaths []string
	filter      func(string) error
	count       int
}

// Option defines a function for passing builder to the NewFileLocator() call
type Option func(*builder)

func newBuilder(opts ...Option) *builder {
	o := &builder{}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = logger.New()
	}
	if o.filter == nil {
		o.filter = assertFile
	}
	return o
}

func (o builder) build() Locator {
	f := file{
		builder: o,
		// Since the `Locate` implementations rely on the root already being specified we update
		// the prefixes to include the root.
		prefixes: getSearchPrefixes(o.root, o.searchPaths...),
	}
	return asRequired(&f)
}

// WithRoot sets the root for the file locator
func WithRoot(root string) Option {
	return func(f *builder) {
		f.root = root
	}
}

// WithLogger sets the logger for the file locator
func WithLogger(logger logger.Interface) Option {
	return func(f *builder) {
		f.logger = logger
	}
}

// WithSearchPaths sets the search paths for the file locator.
func WithSearchPaths(paths ...string) Option {
	return func(f *builder) {
		f.searchPaths = NormalizePaths(paths...)
	}
}

// WithFilter sets the filter for the file locator
// The filter is called for each candidate file and candidates that return nil are considered.
func WithFilter(assert func(string) error) Option {
	return func(f *builder) {
		f.filter = assert
	}
}

// WithCount sets the maximum number of candidates to discover
func WithCount(count int) Option {
	return func(f *builder) {
		f.count = count
	}
}
