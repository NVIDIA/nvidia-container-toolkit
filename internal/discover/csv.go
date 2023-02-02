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

package discover

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

// NewFromCSVFiles creates a discoverer for the specified CSV files. A logger is also supplied.
// The constructed discoverer is comprised of a list, with each element in the list being associated with a
// single CSV files.
func NewFromCSVFiles(logger *logrus.Logger, files []string, driverRoot string) (Discover, error) {
	if len(files) == 0 {
		logger.Warnf("No CSV files specified")
		return None{}, nil
	}

	symlinkLocator := lookup.NewSymlinkLocator(logger, driverRoot)
	locators := map[csv.MountSpecType]lookup.Locator{
		csv.MountSpecDev: lookup.NewCharDeviceLocator(lookup.WithLogger(logger), lookup.WithRoot(driverRoot)),
		csv.MountSpecDir: lookup.NewDirectoryLocator(logger, driverRoot),
		// Libraries and symlinks are handled in the same way
		csv.MountSpecLib: symlinkLocator,
		csv.MountSpecSym: symlinkLocator,
	}

	var mountSpecs []*csv.MountSpec
	for _, filename := range files {
		targets, err := loadCSVFile(logger, filename)
		if err != nil {
			logger.Warnf("Skipping CSV file %v: %v", filename, err)
			continue
		}
		mountSpecs = append(mountSpecs, targets...)
	}

	return newFromMountSpecs(logger, locators, driverRoot, mountSpecs)
}

// loadCSVFile loads the specified CSV file and returns the list of mount specs
func loadCSVFile(logger *logrus.Logger, filename string) ([]*csv.MountSpec, error) {
	// Create a discoverer for each file-kind combination
	targets, err := csv.NewCSVFileParser(logger, filename).Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %v", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	return targets, nil
}

// newFromMountSpecs creates a discoverer for the CSV file. A logger is also supplied.
// A list of csvDiscoverers is returned, with each being associated with a single MountSpecType.
func newFromMountSpecs(logger *logrus.Logger, locators map[csv.MountSpecType]lookup.Locator, driverRoot string, targets []*csv.MountSpec) (Discover, error) {
	if len(targets) == 0 {
		return &None{}, nil
	}

	var discoverers []Discover
	var mountSpecTypes []csv.MountSpecType
	candidatesByType := make(map[csv.MountSpecType][]string)
	for _, t := range targets {
		if _, exists := candidatesByType[t.Type]; !exists {
			mountSpecTypes = append(mountSpecTypes, t.Type)
		}
		candidatesByType[t.Type] = append(candidatesByType[t.Type], t.Path)
	}

	for _, t := range mountSpecTypes {
		locator, exists := locators[t]
		if !exists {
			return nil, fmt.Errorf("no locator defined for '%v'", t)
		}

		var m Discover
		switch t {
		case csv.MountSpecDev:
			m = NewDeviceDiscoverer(logger, locator, driverRoot, candidatesByType[t])
		default:
			m = NewMounts(logger, locator, driverRoot, candidatesByType[t])
		}
		discoverers = append(discoverers, m)

	}

	return &list{discoverers: discoverers}, nil
}
