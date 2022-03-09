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

package csv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultRoot is default location of CSV files that define the modifications required to the OCI spec
	DefaultRoot = "/etc/nvidia-container-runtime/host-files-for-container.d"
)

// GetFileList returns the (non-recursive) list of CSV files in the specified
// folder
func GetFileList(root string) ([]string, error) {
	contents, err := os.ReadDir(root)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read the contents of %v: %v", root, err)
	}

	var csvFilePaths []string
	for _, c := range contents {
		if c.IsDir() {
			continue
		}
		if c.Name() == ".csv" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(c.Name()))
		if ext != ".csv" {
			continue
		}

		csvFilePaths = append(csvFilePaths, filepath.Join(root, c.Name()))
	}

	return csvFilePaths, nil
}

// ParseFile parses the specified file and returns a list of required jetson mounts
func ParseFile(logger *logrus.Logger, filename string) ([]*MountSpec, error) {
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v for reading: %v", filename, err)
	}
	return parseCSVFromReader(logger, csvFile), nil
}

// parseCSVFromReader parses the specified file and returns a list of required jetson mounts
func parseCSVFromReader(logger *logrus.Logger, reader io.Reader) []*MountSpec {
	var targets []*MountSpec

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		target, err := NewMountSpecFromLine(line)
		if err != nil {
			logger.Debugf("Skipping invalid mount spec '%v': %v", line, err)
			continue
		}
		targets = append(targets, target)
	}

	return targets
}
