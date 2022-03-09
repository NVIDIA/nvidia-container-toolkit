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

type csvDiscoverer struct {
	mounts
	filename  string
	mountType csv.MountSpecType
}

var _ Discover = (*csvDiscoverer)(nil)

// NewFromCSV creates a discoverer for the CSV files at the specified root. A logger is also supplied.
func NewFromCSV(logger *logrus.Logger, csvRoot string, root string) (Discover, error) {
	logger.Debugf("Loading CSV files from: %v", csvRoot)

	files, err := csv.GetFileList(csvRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSV file from %v: %v", csvRoot, err)
	}
	if len(files) == 0 {
		logger.Warnf("No CSV files found in %v", csvRoot)
		return None{}, nil
	}

	locators := make(map[csv.MountSpecType]lookup.Locator)
	locators[csv.MountSpecDev] = lookup.NewCharDeviceLocator(logger, root)
	locators[csv.MountSpecDir] = lookup.NewDirectoryLocator(logger, root)
	// Libraries and symlinks are handled in the same way
	symlinkLocator := lookup.NewSymlinkLocator(logger, root)
	locators[csv.MountSpecLib] = symlinkLocator
	locators[csv.MountSpecSym] = symlinkLocator

	var discoverers []Discover
	// Create a discoverer for each file-kind combination
	for _, file := range files {
		targets, err := csv.ParseFile(logger, file)
		if err != nil {
			logger.Warnf("Skipping failed CSV file %v: %v", file, err)
			continue
		}
		if len(targets) == 0 {
			logger.Warnf("Skipping empty CSV file %v", file)
			continue
		}

		candidatesByType := make(map[csv.MountSpecType][]string)
		for _, t := range targets {
			candidatesByType[t.Type] = append(candidatesByType[t.Type], t.Path)
		}

		for t, candidates := range candidatesByType {
			d := csvDiscoverer{
				filename:  file,
				mountType: t,
				mounts: mounts{
					logger:   logger,
					lookup:   locators[t],
					required: candidates,
				},
			}
			discoverers = append(discoverers, &d)
		}

	}

	return &list{discoverers: discoverers}, nil
}

func (d csvDiscoverer) Mounts() ([]Mount, error) {
	if d.mountType == csv.MountSpecDev {
		return d.None.Mounts()
	}

	return d.mounts.Mounts()
}
