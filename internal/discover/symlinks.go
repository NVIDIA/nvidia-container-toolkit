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
	"strings"

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
	mountsFrom              Discover
}

// NewCreateSymlinksHook creates a discoverer for a hook that creates required symlinks in the container
func NewCreateSymlinksHook(logger *logrus.Logger, csvFiles []string, mounts Discover, cfg *Config) (Discover, error) {
	d := symlinks{
		logger:                  logger,
		lookup:                  lookup.NewExecutableLocator(logger, cfg.Root),
		nvidiaCTKExecutablePath: cfg.NVIDIAContainerToolkitCLIExecutablePath,
		csvFiles:                csvFiles,
		mountsFrom:              mounts,
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
		args = append(args, "--csv-filename", f)
	}

	links, err := d.getSpecificLinkArgs()
	if err != nil {
		return nil, fmt.Errorf("failed to determine specific links: %v", err)
	}
	args = append(args, links...)

	h := Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      hookPath,
		Args:      args,
	}

	return []Hook{h}, nil
}

// getSpecificLinkArgs returns the required specic links that need to be created
func (d symlinks) getSpecificLinkArgs() ([]string, error) {
	mounts, err := d.mountsFrom.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover mounts for ldcache update: %v", err)
	}

	linkProcessed := make(map[string]bool)
	var links []string
	for _, m := range mounts {
		var target string
		var link string

		lib := filepath.Base(m.Path)

		if strings.HasPrefix(lib, "libcuda.so") {
			// XXX Many applications wrongly assume that libcuda.so exists (e.g. with dlopen).
			target = "libcuda.so.1"
			link = "libcuda.so"
		} else if strings.HasPrefix(lib, "libGLX_nvidia.so") {
			// XXX GLVND requires this symlink for indirect GLX support.
			target = lib
			link = "libGLX_indirect.so.0"
		} else if strings.HasPrefix(lib, "libnvidia-opticalflow.so") {
			// XXX Fix missing symlink for libnvidia-opticalflow.so.
			target = "libnvidia-opticalflow.so.1"
			link = "libnvidia-opticalflow.so"
		} else {
			continue
		}
		if linkProcessed[link] {
			continue
		}

		linkPath := filepath.Join(filepath.Dir(m.Path), link)
		links = append(links, "--link", fmt.Sprintf("%v:%v", target, linkPath))
		linkProcessed[link] = true
	}

	return links, nil
}
