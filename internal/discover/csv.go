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

// NewFromCSVFiles creates a discoverer for the specified CSV files. A logger is also supplied.
// The constructed discoverer is comprised of a list, with each element in the list being associated with a
// single CSV files.
func NewFromCSVFiles(logger *logrus.Logger, files []string, root string) (Discover, error) {
	if len(files) == 0 {
		logger.Warnf("No CSV files specified")
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
	for _, filename := range files {
		d, err := NewFromCSVFile(logger, locators, filename)
		if err != nil {
			logger.Warnf("Skipping CSV file %v: %v", filename, err)
			continue
		}
		discoverers = append(discoverers, d)
	}

	return &list{discoverers: discoverers}, nil
}

// NewFromCSVFile creates a discoverer for the specified CSV file. A logger is also supplied.
// The constructed discoverer is comprised of a list, with each element in the list being associated with a particular
// MountSpecType.
func NewFromCSVFile(logger *logrus.Logger, locators map[csv.MountSpecType]lookup.Locator, filename string) (Discover, error) {
	// Create a discoverer for each file-kind combination
	targets, err := csv.ParseFile(logger, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %v", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	csvDiscoverers, err := newFromMountSpecs(logger, locators, targets)
	if err != nil {
		return nil, err
	}
	var discoverers []Discover
	for _, d := range csvDiscoverers {
		d.filename = filename
		discoverers = append(discoverers, d)
	}

	return &list{discoverers: discoverers}, nil
}

// newFromMountSpecs creates a discoverer for the CSV file. A logger is also supplied.
// A list of csvDiscoverers is returned, with each being associated with a single MountSpecType.
func newFromMountSpecs(logger *logrus.Logger, locators map[csv.MountSpecType]lookup.Locator, targets []*csv.MountSpec) ([]*csvDiscoverer, error) {
	var discoverers []*csvDiscoverer
	candidatesByType := make(map[csv.MountSpecType][]string)
	for _, t := range targets {
		candidatesByType[t.Type] = append(candidatesByType[t.Type], t.Path)
	}

	for t, candidates := range candidatesByType {
		locator, exists := locators[t]
		if !exists {
			return nil, fmt.Errorf("no locator defined for '%v'", t)
		}
		d := csvDiscoverer{
			mounts: mounts{
				logger:   logger,
				lookup:   locator,
				required: candidates,
			},
			mountType: t,
		}
		discoverers = append(discoverers, &d)
	}

	return discoverers, nil
}

// Mounts returns the discovered mounts for the csvDiscoverer.
// Note that if the discoverer is for the device MountSpecType, the list of mounts is empty.
func (d csvDiscoverer) Mounts() ([]Mount, error) {
	if d.mountType == csv.MountSpecDev {
		return d.None.Mounts()
	}

	return d.mounts.Mounts()
}

// Devices returns the discovered devices for the csvDiscoverer.
// Note that if the discoverer is not for the device MountSpecType, the list of devices is empty.
func (d csvDiscoverer) Devices() ([]Device, error) {
	if d.mountType != csv.MountSpecDev {
		return d.None.Devices()
	}

	mounts, err := d.mounts.Mounts()
	if err != nil {
		return nil, err
	}
	var devices []Device
	for _, mount := range mounts {
		device := Device(mount)
		devices = append(devices, device)
	}

	return devices, nil
}
