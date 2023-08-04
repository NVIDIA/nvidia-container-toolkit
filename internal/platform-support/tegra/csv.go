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

package tegra

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

// newDiscovererFromCSVFiles creates a discoverer for the specified CSV files. A logger is also supplied.
// The constructed discoverer is comprised of a list, with each element in the list being associated with a
// single CSV files.
func newDiscovererFromCSVFiles(logger logger.Interface, files []string, driverRoot string, nvidiaCTKPath string, librarySearchPaths []string) (discover.Discover, error) {
	if len(files) == 0 {
		logger.Warningf("No CSV files specified")
		return discover.None{}, nil
	}

	targetsByType := getTargetsFromCSVFiles(logger, files)

	devices := discover.NewDeviceDiscoverer(
		logger,
		lookup.NewCharDeviceLocator(lookup.WithLogger(logger), lookup.WithRoot(driverRoot)),
		driverRoot,
		targetsByType[csv.MountSpecDev],
	)

	directories := discover.NewMounts(
		logger,
		lookup.NewDirectoryLocator(lookup.WithLogger(logger), lookup.WithRoot(driverRoot)),
		driverRoot,
		targetsByType[csv.MountSpecDir],
	)

	// Libraries and symlinks use the same locator.
	searchPaths := append(librarySearchPaths, "/")
	symlinkLocator := lookup.NewSymlinkLocator(
		lookup.WithLogger(logger),
		lookup.WithRoot(driverRoot),
		lookup.WithSearchPaths(searchPaths...),
	)
	libraries := discover.NewMounts(
		logger,
		symlinkLocator,
		driverRoot,
		targetsByType[csv.MountSpecLib],
	)

	nonLibSymlinks := ignoreFilenamePatterns{"*.so", "*.so.[0-9]"}.Apply(targetsByType[csv.MountSpecSym]...)
	logger.Debugf("Non-lib symlinks: %v", nonLibSymlinks)
	symlinks := discover.NewMounts(
		logger,
		symlinkLocator,
		driverRoot,
		nonLibSymlinks,
	)
	createSymlinks := createCSVSymlinkHooks(logger, nonLibSymlinks, libraries, nvidiaCTKPath)

	d := discover.Merge(
		devices,
		directories,
		libraries,
		symlinks,
		createSymlinks,
	)

	return d, nil
}

// getTargetsFromCSVFiles returns the list of mount specs from the specified CSV files.
// These are aggregated by mount spec type.
func getTargetsFromCSVFiles(logger logger.Interface, files []string) map[csv.MountSpecType][]string {
	targetsByType := make(map[csv.MountSpecType][]string)
	for _, filename := range files {
		targets, err := loadCSVFile(logger, filename)
		if err != nil {
			logger.Warningf("Skipping CSV file %v: %v", filename, err)
			continue
		}
		for _, t := range targets {
			targetsByType[t.Type] = append(targetsByType[t.Type], t.Path)
		}
	}
	return targetsByType
}

// loadCSVFile loads the specified CSV file and returns the list of mount specs
func loadCSVFile(logger logger.Interface, filename string) ([]*csv.MountSpec, error) {
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
