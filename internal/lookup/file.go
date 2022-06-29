/*
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
*/

package lookup

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// file can be used to locate file (or file-like elements) at a specified set of
// prefixes. The validity of a file is determined by a filter function.
type file struct {
	logger   *log.Logger
	root     string
	prefixes []string
	filter   func(string) error
}

// NewFileLocator creates a Locator that can be used to find files at the specified root. A logger
// can also be specified.
func NewFileLocator(logger *log.Logger, root string) Locator {
	l := newFileLocator(logger, root)

	return &l
}

func newFileLocator(logger *log.Logger, root string) file {
	return file{
		logger:   logger,
		prefixes: []string{root},
		filter:   assertFile,
	}
}

var _ Locator = (*file)(nil)

// Locate attempts to find files with names matching the specified pattern.
// All prefixes are searched and any matching candidates are returned. If no matches are found, an error is returned.
func (p file) Locate(pattern string) ([]string, error) {
	var filenames []string
	for _, prefix := range p.prefixes {
		pathPattern := filepath.Join(prefix, pattern)
		candidates, err := filepath.Glob(pathPattern)
		if err != nil {
			p.logger.Debugf("Checking pattern '%v' failed: %v", pathPattern, err)
		}

		for _, candidate := range candidates {
			p.logger.Debugf("Checking candidate '%v'", candidate)
			err := p.filter(candidate)
			if err != nil {
				p.logger.Debugf("Candidate '%v' does not meet requirements: %v", candidate, err)
				continue
			}
			filenames = append(filenames, candidate)
		}
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("pattern %v not found", pattern)
	}
	return filenames, nil
}

// Relative returns the path relative to the root for the file locator
func (p file) Relative(path string) (string, error) {
	if p.root == "" || p.root == "/" {
		return path, nil
	}

	return filepath.Rel(p.root, path)
}

// assertFile checks whether the specified path is a regular file
func assertFile(filename string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("error getting info for %v: %v", filename, err)
	}

	if info.IsDir() {
		return fmt.Errorf("specified path '%v' is a directory", filename)
	}

	return nil
}
