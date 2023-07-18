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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
)

type symlinkHook struct {
	None
	logger        logger.Interface
	driverRoot    string
	nvidiaCTKPath string
	csvFiles      []string
	mountsFrom    Discover
}

// NewCreateSymlinksHook creates a discoverer for a hook that creates required symlinks in the container
func NewCreateSymlinksHook(logger logger.Interface, csvFiles []string, mounts Discover, nvidiaCTKPath string) (Discover, error) {
	d := symlinkHook{
		logger:        logger,
		nvidiaCTKPath: nvidiaCTKPath,
		csvFiles:      csvFiles,
		mountsFrom:    mounts,
	}

	return &d, nil
}

// Hooks returns a hook to create the symlinks from the required CSV files
func (d symlinkHook) Hooks() ([]Hook, error) {
	specificLinks, err := d.getSpecificLinks()
	if err != nil {
		return nil, fmt.Errorf("failed to determine specific links: %v", err)
	}

	csvSymlinks := d.getCSVFileSymlinks()
	var args []string
	for _, link := range append(csvSymlinks, specificLinks...) {
		args = append(args, "--link", link)
	}

	hook := CreateNvidiaCTKHook(
		d.nvidiaCTKPath,
		"create-symlinks",
		args...,
	)

	return []Hook{hook}, nil
}

// getSpecificLinks returns the required specic links that need to be created
func (d symlinkHook) getSpecificLinks() ([]string, error) {
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
		linkProcessed[link] = true

		linkPath := filepath.Join(filepath.Dir(m.Path), link)
		links = append(links, fmt.Sprintf("%v::%v", target, linkPath))
	}

	return links, nil
}

func (d symlinkHook) getCSVFileSymlinks() []string {
	chainLocator := lookup.NewSymlinkChainLocator(
		lookup.WithLogger(d.logger),
		lookup.WithRoot(d.driverRoot),
	)

	var candidates []string
	for _, file := range d.csvFiles {
		mountSpecs, err := csv.NewCSVFileParser(d.logger, file).Parse()
		if err != nil {
			d.logger.Debugf("Skipping CSV file %v: %v", file, err)
			continue
		}

		for _, ms := range mountSpecs {
			if ms.Type != csv.MountSpecSym {
				continue
			}
			targets, err := chainLocator.Locate(ms.Path)
			if err != nil {
				d.logger.Warningf("Failed to locate symlink %v", ms.Path)
			}
			candidates = append(candidates, targets...)
		}
	}

	var links []string
	created := make(map[string]bool)
	// candidates is a list of absolute paths to symlinks in a chain, or the final target of the chain.
	for _, candidate := range candidates {
		target, err := symlinks.Resolve(candidate)
		if err != nil {
			d.logger.Debugf("Skipping invalid link: %v", err)
			continue
		} else if target == candidate {
			d.logger.Debugf("%v is not a symlink", candidate)
			continue
		}

		link := fmt.Sprintf("%v::%v", target, candidate)
		if created[link] {
			d.logger.Debugf("skipping duplicate link: %v", link)
			continue
		}
		created[link] = true

		links = append(links, link)
	}

	return links
}
