/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package nvcdi

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/dxcore"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

const (
	libcudaSo = "libcuda.so.1.1"
)

var dxcoreLibraries = []string{
	"libdxcore.so", /* Core library for dxcore support */
}

var requiredDriverStoreFiles = []string{
	"libcuda.so.1.1",                /* Core library for cuda support */
	"libcuda_loader.so",             /* Core library for cuda support on WSL */
	"libnvidia-ptxjitcompiler.so.1", /* Core library for PTX Jit support */
	"libnvidia-ml.so.1",             /* Core library for nvml */
	"libnvidia-ml_loader.so",        /* Core library for nvml on WSL */
	"libnvdxgdmal.so.1",             /* dxgdmal library for cuda */
	"nvcubins.bin",                  /* Binary containing GPU code for cuda */
	"nvidia-smi",                    /* nvidia-smi binary*/
}

// newWSLDriverDiscoverer returns a Discoverer for WSL2 drivers.
func (l *wsllib) newWSLDriverDiscoverer() (discover.Discover, error) {
	if err := dxcore.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize dxcore: %w", err)
	}
	defer func() {
		if err := dxcore.Shutdown(); err != nil {
			l.logger.Warningf("failed to shutdown dxcore: %w", err)
		}
	}()

	driverStorePaths := dxcore.GetDriverStorePaths()
	if len(driverStorePaths) == 0 {
		return nil, fmt.Errorf("no driver store paths found")
	}
	if len(driverStorePaths) > 1 {
		l.logger.Warningf("Found multiple driver store paths: %v", driverStorePaths)
	}

	nvDriverStorePath, err := l.getNVIDIADriverStorePath(driverStorePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to find NVIDIA driver store path: %w", err)
	}

	l.logger.Infof("Using WSL driver store path: %v", nvDriverStorePath)

	dxcoreMounts := discover.NewMounts(
		l.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(l.logger),
			lookup.WithSearchPaths(
				"/usr/lib/wsl/lib",
			),
			lookup.WithCount(1),
		),
		l.driver.Root,
		dxcoreLibraries,
	)

	requiredDriverStoreMounts := discover.NewMounts(
		l.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(l.logger),
			lookup.WithSearchPaths(
				nvDriverStorePath,
			),
			lookup.WithCount(1),
		),
		l.driver.Root,
		requiredDriverStoreFiles,
	)

	additionalDriverStoreMounts, err := l.getAdditionalMountsFromDriverStore(nvDriverStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional mounts from driver store: %w", err)
	}

	symlinkHook := nvidiaSMISimlinkHook{
		logger:      l.logger,
		mountsFrom:  requiredDriverStoreMounts,
		hookCreator: l.hookCreator,
	}

	ldcacheHook, _ := discover.NewLDCacheUpdateHook(l.logger, discover.Merge(requiredDriverStoreMounts, dxcoreMounts), l.hookCreator)

	d := discover.Merge(
		dxcoreMounts,
		requiredDriverStoreMounts,
		additionalDriverStoreMounts,
		symlinkHook,
		ldcacheHook,
	)

	return d, nil
}

// getNVIDIADriverStorePath returns the driver store path associated with NVIDIA GPUs
func (l *wsllib) getNVIDIADriverStorePath(driverStorePaths []string) (string, error) {
	fileLocator := lookup.NewFileLocator(
		lookup.WithLogger(l.logger),
		lookup.WithSearchPaths(
			driverStorePaths...,
		),
		lookup.WithCount(1),
	)
	matches, err := fileLocator.Locate(libcudaSo)
	if err != nil {
		return "", fmt.Errorf("failed to locate %s at WSL driver store paths: %w", libcudaSo, err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("could not locate %s at WSL driver store paths", libcudaSo)
	}

	return filepath.Dir(matches[0]), nil
}

// getAdditionalMountsFromDriverStore discovers additional NVIDIA libraries (.so files) from the
// driver store that are not in the required list of libraries.
func (l *wsllib) getAdditionalMountsFromDriverStore(driverStore string) (discover.Discover, error) {
	additionalLibs, err := l.getAdditionalFilesFromDriverStore(driverStore, requiredDriverStoreFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup additional files in driver store: %w", err)
	}

	mounts := discover.NewMounts(
		l.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(l.logger),
			lookup.WithRoot(l.driver.Root),
		),
		l.driver.Root,
		additionalLibs,
	)

	return mounts, nil
}

func (l *wsllib) getAdditionalFilesFromDriverStore(driverStore string, excludeFiles []string) ([]string, error) {
	fileLocator := lookup.AsOptional(
		lookup.NewFileLocator(
			lookup.WithLogger(l.logger),
			lookup.WithSearchPaths(driverStore),
			lookup.WithFilter(func(s string) error {
				if slices.Contains(excludeFiles, filepath.Base(s)) {
					return fmt.Errorf("file %s is excluded", s)
				}
				return nil
			}),
		))

	libs, err := fileLocator.Locate("*.so*")
	if err != nil {
		return nil, fmt.Errorf("failed to find additional '.so' files in driver store: %w", err)
	}
	bins, err := fileLocator.Locate("*.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to find additional '.bin' files in driver store: %w", err)
	}
	dlls, err := fileLocator.Locate("*.dll")
	if err != nil {
		return nil, fmt.Errorf("failed to find additional '.dll' files in driver store: %w", err)
	}
	return slices.Concat(libs, bins, dlls), nil
}

type nvidiaSMISimlinkHook struct {
	discover.None
	logger      logger.Interface
	mountsFrom  discover.Discover
	hookCreator discover.HookCreator
}

// Hooks returns a hook that creates a symlink to nvidia-smi in the driver store.
// On WSL2 the driver store location is used unchanged, for this reason we need
// to create a symlink from /usr/bin/nvidia-smi to the nvidia-smi binary in the
// driver store.
func (m nvidiaSMISimlinkHook) Hooks() ([]discover.Hook, error) {
	mounts, err := m.mountsFrom.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover mounts: %w", err)
	}

	var target string
	for _, mount := range mounts {
		if filepath.Base(mount.Path) == "nvidia-smi" {
			target = mount.Path
			break
		}
	}

	if target == "" {
		m.logger.Warningf("Failed to find nvidia-smi in mounts: %v", mounts)
		return nil, nil
	}
	link := "/usr/bin/nvidia-smi"
	links := []string{fmt.Sprintf("%s::%s", target, link)}
	symlinkHook := m.hookCreator.Create(CreateSymlinksHook, links...)

	return symlinkHook.Hooks()
}
