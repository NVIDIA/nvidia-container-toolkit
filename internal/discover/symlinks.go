/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/sirupsen/logrus"
)

type symlinks struct {
	None
	logger                  *logrus.Logger
	lookup                  lookup.Locator
	nvidiaCTKExecutablePath string
	csvFiles                []string
}

// NewCreateSymlinksHook creates a discoverer for a hook that creates required symlinks in the container
func NewCreateSymlinksHook(logger *logrus.Logger, csvFiles []string, cfg *Config) (Discover, error) {
	d := symlinks{
		logger:                  logger,
		lookup:                  lookup.NewExecutableLocator(logger, cfg.Root),
		nvidiaCTKExecutablePath: cfg.NVIDIAContainerToolkitCLIExecutablePath,
		csvFiles:                csvFiles,
	}

	return &d, nil
}

// Hooks returns a hook to create the symlinks from the required CSV files
func (d symlinks) Hooks() ([]Hook, error) {
	hookPath := nvidiaCTKDefaultFilePath
	targets, err := d.lookup.Locate(d.nvidiaCTKExecutablePath)
	if err != nil {
		d.logger.Warnf("Failed to locate %v: %v", d.nvidiaCTKExecutablePath, err)
	} else if len(targets) == 0 {
		d.logger.Warnf("%v not found", d.nvidiaCTKExecutablePath)
	} else {
		d.logger.Debugf("Found %v candidates: %v", d.nvidiaCTKExecutablePath, targets)
		hookPath = targets[0]
	}
	d.logger.Debugf("Using NVIDIA Container Toolkit CLI path %v", hookPath)

	args := []string{hookPath, "hook", "create-symlinks"}
	for _, f := range d.csvFiles {
		args = append(args, "--csv-filenames", f)
	}

	h := Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      hookPath,
		Args:      args,
	}

	return []Hook{h}, nil
}
