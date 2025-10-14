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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

func (o options) newDiscovererFromMountSpecs() (discover.Discover, error) {
	pathsByType := o.MountSpecPathsByType()
	if len(pathsByType) == 0 {
		o.logger.Warningf("No mount specs specified")
		return discover.None{}, nil
	}

	devices := discover.NewCharDeviceDiscoverer(
		o.logger,
		o.devRoot,
		pathsByType[csv.MountSpecDev],
	)

	directories := discover.NewMounts(
		o.logger,
		lookup.NewDirectoryLocator(lookup.WithLogger(o.logger), lookup.WithRoot(o.driverRoot)),
		o.driverRoot,
		pathsByType[csv.MountSpecDir],
	)

	// We create a discoverer for mounted libraries and add additional .so
	// symlinks for the driver.
	libraries := discover.WithDriverDotSoSymlinks(
		o.logger,
		discover.NewMounts(
			o.logger,
			o.symlinkLocator,
			o.driverRoot,
			pathsByType[csv.MountSpecLib],
		),
		"",
		o.hookCreator,
	)

	// We process the explicitly requested symlinks.
	symlinkTargets := pathsByType[csv.MountSpecSym]
	o.logger.Debugf("Filtered symlink targets: %v", symlinkTargets)
	symlinks := discover.NewMounts(
		o.logger,
		o.symlinkLocator,
		o.driverRoot,
		symlinkTargets,
	)
	createSymlinks := o.createCSVSymlinkHooks(symlinkTargets)

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
func MountSpecsFromCSVFiles(logger logger.Interface, csvFiles ...string) MountSpecPathsByTyper {
	var tts []MountSpecPathsByTyper

	for _, filename := range csvFiles {
		tts = append(tts, &fromCSVFile{logger, filename})
	}
	return Merge(tts...)
}

type fromCSVFile struct {
	logger   logger.Interface
	filename string
}

// MountSpecPathsByType returns mountspecs defined in the specified CSV file.
func (t *fromCSVFile) MountSpecPathsByType() MountSpecPathsByType {
	// Create a discoverer for each file-kind combination
	targets, err := csv.NewCSVFileParser(t.logger, t.filename).Parse()
	if err != nil {
		t.logger.Warningf("failed to parse CSV file %v: %v", t.filename, err)
		return nil
	}

	targetsByType := make(MountSpecPathsByType)
	for _, t := range targets {
		targetsByType[t.Type] = append(targetsByType[t.Type], t.Path)
	}
	return targetsByType
}
