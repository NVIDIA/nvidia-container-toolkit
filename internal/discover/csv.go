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

// charDevices is a discover for a list of character devices
type charDevices mounts

var _ Discover = (*charDevices)(nil)

// NewFromCSVFiles creates a discoverer for the specified CSV files. A logger is also supplied.
// The constructed discoverer is comprised of a list, with each element in the list being associated with a
// single CSV files.
func NewFromCSVFiles(logger *logrus.Logger, files []string, root string) (Discover, error) {
	if len(files) == 0 {
		logger.Warnf("No CSV files specified")
		return None{}, nil
	}

	symlinkLocator := lookup.NewSymlinkLocator(logger, root)
	locators := map[csv.MountSpecType]lookup.Locator{
		csv.MountSpecDev: lookup.NewCharDeviceLocator(logger, root),
		csv.MountSpecDir: lookup.NewDirectoryLocator(logger, root),
		// Libraries and symlinks are handled in the same way
		csv.MountSpecLib: symlinkLocator,
		csv.MountSpecSym: symlinkLocator,
	}

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
	targets, err := csv.NewCSVFileParser(logger, filename).Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %v", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	return newFromMountSpecs(logger, locators, targets)
}

// newFromMountSpecs creates a discoverer for the CSV file. A logger is also supplied.
// A list of csvDiscoverers is returned, with each being associated with a single MountSpecType.
func newFromMountSpecs(logger *logrus.Logger, locators map[csv.MountSpecType]lookup.Locator, targets []*csv.MountSpec) (Discover, error) {
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

		m := &mounts{
			logger:   logger,
			lookup:   locator,
			required: candidatesByType[t],
		}

		switch t {
		case csv.MountSpecDev:
			// For device mount specs, we insert a charDevices into the list of discoverers.
			discoverers = append(discoverers, (*charDevices)(m))
		default:
			discoverers = append(discoverers, m)
		}
	}

	return &list{discoverers: discoverers}, nil
}

// Mounts returns the discovered mounts for the charDevices. Since this explicitly specifies a
// device list, the mounts are nil.
func (d *charDevices) Mounts() ([]Mount, error) {
	return nil, nil
}

// Devices returns the discovered devices for the charDevices. Here the device nodes are first
// discovered as mounts and these are converted to devices.
func (d *charDevices) Devices() ([]Device, error) {
	devicesAsMounts, err := (*mounts)(d).Mounts()
	if err != nil {
		return nil, err
	}
	var devices []Device
	for _, mount := range devicesAsMounts {
		devices = append(devices, Device(mount))
	}

	return devices, nil
}
