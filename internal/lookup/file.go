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

type file struct {
	logger   *log.Logger
	prefixes []string
	filter   func(string) error
}

func NewFileLocator(root string) Locator {
	return NewFileLocatorWithLogger(log.StandardLogger(), root)
}

func NewFileLocatorWithLogger(logger *log.Logger, root string) Locator {
	l := file{
		logger:   logger,
		prefixes: []string{root},
		filter:   assertFile,
	}

	return &l
}

var _ Locator = (*file)(nil)

func (p file) Locate(filename string) ([]string, error) {
	var filenames []string
	for _, prefix := range p.prefixes {
		candidate := filepath.Join(prefix, filename)
		p.logger.Debugf("Checking candidate '%v'", candidate)
		err := p.filter(candidate)
		if err != nil {
			p.logger.Debugf("Candidate '%v' does not meet requirements: %v", candidate, err)
			continue
		}
		filenames = append(filenames, candidate)
	}
	if len(filename) == 0 {
		return nil, fmt.Errorf("file %v not found", filename)
	}
	return filenames, nil
}

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
