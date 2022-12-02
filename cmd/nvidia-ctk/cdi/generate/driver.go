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

package generate

import (
	"fmt"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

type driverLibraries struct {
	logger    *logrus.Logger
	root      string
	libraries []string
}

var _ discover.Discover = (*driverLibraries)(nil)

// NewDriverDiscoverer creates a discoverer for the libraries and binaries associated with a driver installation.
// The supplied NVML Library is used to query the expected driver version.
func NewDriverDiscoverer(logger *logrus.Logger, root string, nvmllib nvml.Interface) (discover.Discover, error) {
	libraries, err := NewDriverLibraryDiscoverer(logger, root, nvmllib)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for driver libraries: %v", err)
	}

	binaries := discover.NewMounts(
		logger,
		lookup.NewExecutableLocator(logger, root),
		root,
		[]string{
			"nvidia-smi",              /* System management interface */
			"nvidia-debugdump",        /* GPU coredump utility */
			"nvidia-persistenced",     /* Persistence mode utility */
			"nvidia-cuda-mps-control", /* Multi process service CLI */
			"nvidia-cuda-mps-server",  /* Multi process service server */
		},
	)

	d := discover.Merge(
		libraries,
		binaries,
	)

	return d, nil
}

// NewDriverLibraryDiscoverer creates a discoverer for the libraries associated with the specified driver version.
func NewDriverLibraryDiscoverer(logger *logrus.Logger, root string, nvmllib nvml.Interface) (discover.Discover, error) {
	version, r := nvmllib.SystemGetDriverVersion()
	if r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to determine driver version: %v", r)
	}

	libraries, err := findVersionLibs(logger, root, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries for driver version: %v", r)
	}

	d := driverLibraries{
		logger:    logger,
		root:      root,
		libraries: libraries,
	}

	return &d, nil
}

// Devices are empty for this discoverer
func (d *driverLibraries) Devices() ([]discover.Device, error) {
	return nil, nil
}

// Mounts returns the mounts for the driver libraries
func (d *driverLibraries) Mounts() ([]discover.Mount, error) {
	var mounts []discover.Mount
	for _, d := range d.libraries {
		mount := discover.Mount{
			HostPath: d,
			Path:     d,
		}
		mounts = append(mounts, mount)
	}

	return mounts, nil
}

// Hooks returns a hook that updates the LDCache for the specified driver library paths.
func (d *driverLibraries) Hooks() ([]discover.Hook, error) {
	locator := lookup.NewExecutableLocator(d.logger, d.root)

	hook := discover.CreateLDCacheUpdateHook(
		d.logger,
		locator,
		nvidiaCTKExecutable,
		nvidiaCTKDefaultFilePath,
		d.libraries,
	)

	return []discover.Hook{hook}, nil
}

func findVersionLibs(logger *logrus.Logger, root string, version string) ([]string, error) {
	logger.Infof("Using driver version %v", version)

	cache, err := ldcache.New(logger, root)
	if err != nil {
		return nil, fmt.Errorf("failed to load ldcache: %v", err)
	}

	libs32, libs64 := cache.List()

	var libs []string
	for _, l := range libs64 {
		if strings.HasSuffix(l, version) {
			logger.Infof("found 64-bit driver lib: %v", l)
			libs = append(libs, l)
		}
	}

	for _, l := range libs32 {
		if strings.HasSuffix(l, version) {
			logger.Infof("found 32-bit driver lib: %v", l)
			libs = append(libs, l)
		}
	}

	return libs, nil
}
