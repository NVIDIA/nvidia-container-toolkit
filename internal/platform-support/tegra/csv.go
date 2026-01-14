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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

// newDiscovererFromMountSpecs creates a discoverer for the specified mount specs.
func (o options) newDiscovererFromMountSpecs(targetsByType MountSpecPathsByType) (discover.Discover, error) {
	if len(targetsByType) == 0 {
		o.logger.Warningf("No mount specs specified")
		return discover.None{}, nil
	}

	devices := discover.NewCharDeviceDiscoverer(
		o.logger,
		o.devRoot,
		targetsByType[csv.MountSpecDev],
	)

	directories := discover.NewMounts(
		o.logger,
		lookup.NewDirectoryLocator(lookup.WithLogger(o.logger), lookup.WithRoot(o.driverRoot)),
		o.driverRoot,
		targetsByType[csv.MountSpecDir],
	)

	// We create a discoverer for mounted libraries and add additional .so
	// symlinks for the driver.
	libraries := discover.WithDriverDotSoSymlinks(
		o.logger,
		discover.NewMounts(
			o.logger,
			o.symlinkLocator,
			o.driverRoot,
			targetsByType[csv.MountSpecLib],
		),
		"",
		o.hookCreator,
	)

	// We process the explicitly requested symlinks.
	symlinks := discover.NewMounts(
		o.logger,
		o.symlinkLocator,
		o.driverRoot,
		targetsByType[csv.MountSpecSym],
	)
	createSymlinks := o.createCSVSymlinkHooks(targetsByType[csv.MountSpecSym])

	d := discover.Merge(
		devices,
		directories,
		libraries,
		symlinks,
		createSymlinks,
	)

	return d, nil
}

// MountSpecsFromCSVFiles returns a MountSpecPathsByTyper for the specified list
// of CSV files.
func MountSpecsFromCSVFiles(logger logger.Interface, csvFilePaths ...string) MountSpecPathsByType {
	var mountSpecs mountSpecPathsByTypers

	for _, filename := range csvFilePaths {
		targets, err := loadCSVFile(logger, filename)
		if err != nil {
			logger.Warningf("Skipping CSV file %v: %v", filename, err)
			continue
		}
		targetsByType := make(MountSpecPathsByType)
		for _, t := range targets {
			targetsByType[t.Type] = append(targetsByType[t.Type], t.Path)
		}
		mountSpecs = append(mountSpecs, targetsByType)
	}
	return mountSpecs.MountSpecPathsByType()
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
