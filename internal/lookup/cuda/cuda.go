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

package cuda

import (
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

type cudaLocator struct {
	logger     *logrus.Logger
	driverRoot string
}

// Options is a function that configures a cudaLocator.
type Options func(*cudaLocator)

// WithLogger is an option that configures the logger used by the locator.
func WithLogger(logger *logrus.Logger) Options {
	return func(c *cudaLocator) {
		c.logger = logger
	}
}

// WithDriverRoot is an option that configures the driver root used by the locator.
func WithDriverRoot(driverRoot string) Options {
	return func(c *cudaLocator) {
		c.driverRoot = driverRoot
	}
}

// New creates a new CUDA library locator.
func New(opts ...Options) lookup.Locator {
	c := &cudaLocator{}
	for _, opt := range opts {
		opt(c)
	}

	if c.logger == nil {
		c.logger = logrus.StandardLogger()
	}
	if c.driverRoot == "" {
		c.driverRoot = "/"
	}

	return c
}

// Locate returns the path to the libcuda.so.RMVERSION file.
// libcuda.so is prefixed to the specified pattern.
func (l *cudaLocator) Locate(pattern string) ([]string, error) {
	ldcacheLocator, err := lookup.NewLibraryLocator(
		l.logger,
		l.driverRoot,
	)
	if err != nil {
		l.logger.Debugf("Failed to create LDCache locator: %v", err)
	}

	fullPattern := "libcuda.so" + pattern

	candidates, err := ldcacheLocator.Locate("libcuda.so")
	if err == nil {
		for _, c := range candidates {
			if match, err := filepath.Match(fullPattern, filepath.Base(c)); err != nil || !match {
				l.logger.Debugf("Skipping non-matching candidate %v: %v", c, err)
				continue
			}
			return []string{c}, nil
		}
	}
	l.logger.Debugf("Could not locate %q in LDCache: Checking predefined library paths.", pattern)

	pathLocator := lookup.NewFileLocator(
		lookup.WithLogger(l.logger),
		lookup.WithRoot(l.driverRoot),
		lookup.WithSearchPaths(
			"/usr/lib64",
			"/usr/lib/x86_64-linux-gnu",
			"/usr/lib/aarch64-linux-gnu",
			"/usr/lib/x86_64-linux-gnu/nvidia/current",
			"/usr/lib/aarch64-linux-gnu/nvidia/current",
		),
		lookup.WithCount(1),
	)

	return pathLocator.Locate(fullPattern)
}
