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
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/sirupsen/logrus"
)

// NewLDCacheUpdateHook creates a discoverer that updates the ldcache for the specified mounts. A logger can also be specified
func NewLDCacheUpdateHook(logger *logrus.Logger, mounts Discover, cfg *Config) (Discover, error) {
	d := ldconfig{
		logger:                  logger,
		mountsFrom:              mounts,
		lookup:                  lookup.NewExecutableLocator(logger, cfg.Root),
		nvidiaCTKExecutablePath: cfg.NVIDIAContainerToolkitCLIExecutablePath,
	}

	return &d, nil
}

const (
	nvidiaCTKDefaultFilePath = "/usr/bin/nvidia-ctk"
)

type ldconfig struct {
	None
	logger                  *logrus.Logger
	mountsFrom              Discover
	lookup                  lookup.Locator
	nvidiaCTKExecutablePath string
}

// Hooks checks the required mounts for libraries and returns a hook to update the LDcache for the discovered paths.
func (d ldconfig) Hooks() ([]Hook, error) {
	mounts, err := d.mountsFrom.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover mounts for ldcache update: %v", err)
	}

	libDirs := getLibDirs(mounts)

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

	args := []string{hookPath, "hook", "update-ldcache"}
	for _, f := range libDirs {
		args = append(args, "--folder", f)
	}
	h := Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      hookPath,
		Args:      args,
	}

	return []Hook{h}, nil
}

// getLibDirs extracts the library dirs from the specified mounts
func getLibDirs(mounts []Mount) []string {
	var paths []string
	checked := make(map[string]bool)

	for _, m := range mounts {
		dir := filepath.Dir(m.Path)
		if dir == "" {
			continue
		}

		_, exists := checked[dir]
		if exists {
			continue
		}
		checked[dir] = isLibName(filepath.Base(m.Path))

		if checked[dir] {
			paths = append(paths, dir)
		}
	}

	sort.Strings(paths)

	return paths
}

// isLibName checks if the specified filename is a library (i.e. ends in `.so*`)
func isLibName(filename string) bool {
	parts := strings.Split(filename, ".")

	for _, p := range parts {
		if p == "so" {
			return true
		}
	}

	return false
}
