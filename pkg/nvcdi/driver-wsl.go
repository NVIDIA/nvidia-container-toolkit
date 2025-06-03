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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/dxcore"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

var requiredDriverStoreFiles = []string{
	"libcuda.so.1.1",                /* Core library for cuda support */
	"libcuda_loader.so",             /* Core library for cuda support on WSL */
	"libnvidia-ptxjitcompiler.so.1", /* Core library for PTX Jit support */
	"libnvidia-ml.so.1",             /* Core library for nvml */
	"libnvidia-ml_loader.so",        /* Core library for nvml on WSL */
	"libdxcore.so",                  /* Core library for dxcore support */
	"libnvdxgdmal.so.1",             /* dxgdmal library for cuda */
	"nvcubins.bin",                  /* Binary containing GPU code for cuda */
	"nvidia-smi",                    /* nvidia-smi binary*/
}

// newWSLDriverDiscoverer returns a Discoverer for WSL2 drivers.
func newWSLDriverDiscoverer(logger logger.Interface, driverRoot string, hookCreator discover.HookCreator, ldconfigPath string) (discover.Discover, error) {
	if err := dxcore.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize dxcore: %w", err)
	}
	defer func() {
		if err := dxcore.Shutdown(); err != nil {
			logger.Warningf("failed to shutdown dxcore: %w", err)
		}
	}()

	driverStorePaths := dxcore.GetDriverStorePaths()
	if len(driverStorePaths) == 0 {
		return nil, fmt.Errorf("no driver store paths found")
	}
	if len(driverStorePaths) > 1 {
		logger.Warningf("Found multiple driver store paths: %v", driverStorePaths)
	}
	logger.Infof("Using WSL driver store paths: %v", driverStorePaths)

	driverStorePaths = append(driverStorePaths, "/usr/lib/wsl/lib")

	driverStoreMounts := discover.NewMounts(
		logger,
		lookup.NewFileLocator(
			lookup.WithLogger(logger),
			lookup.WithSearchPaths(
				driverStorePaths...,
			),
			lookup.WithCount(1),
		),
		driverRoot,
		requiredDriverStoreFiles,
	)

	symlinkHook := nvidiaSMISimlinkHook{
		logger:      logger,
		mountsFrom:  driverStoreMounts,
		hookCreator: hookCreator,
	}

	ldcacheHook, _ := discover.NewLDCacheUpdateHook(logger, driverStoreMounts, hookCreator, ldconfigPath)

	d := discover.Merge(
		driverStoreMounts,
		symlinkHook,
		ldcacheHook,
	)

	return d, nil
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
