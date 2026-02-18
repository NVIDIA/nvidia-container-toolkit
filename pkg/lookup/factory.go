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

// Factory defines a builder for locators.
type Factory struct {
	logger      logger.Interface
	root        string
	searchPaths []string
	filter      func(string) error
	count       int
}

type Option func(*Factory)

func NewFactory(opts ...Option) *Factory {
	o := &Factory{}
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

// WithRoot sets the root for the file locator
func WithRoot(root string) Option {
	return func(f *Factory) {
		f.root = root
	}
}

// WithLogger sets the logger for the file locator
func WithLogger(logger logger.Interface) Option {
	return func(f *Factory) {
		f.logger = logger
	}
}

// WithSearchPaths sets the search paths for the file locator.
func WithSearchPaths(paths ...string) Option {
	return func(f *Factory) {
		f.searchPaths = NormalizePaths(paths...)
	}
}

// WithFilter sets the filter for the file locator
// The filter is called for each candidate file and candidates that return nil are considered.
func WithFilter(assert func(string) error) Option {
	return func(f *Factory) {
		f.filter = assert
	}
}

// WithCount sets the maximum number of candidates to discover
func WithCount(count int) Option {
	return func(f *Factory) {
		f.count = count
	}
}
