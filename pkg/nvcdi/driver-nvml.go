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
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/cuda"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// NewDriverDiscoverer creates a discoverer for the libraries and binaries associated with a driver installation.
// The supplied NVML Library is used to query the expected driver version.
func (l *nvmllib) NewDriverDiscoverer() (discover.Discover, error) {
	if r := l.nvmllib.Init(); r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to initialize NVML: %v", r)
	}
	defer func() {
		if r := l.nvmllib.Shutdown(); r != nvml.SUCCESS {
			l.logger.Warningf("failed to shutdown NVML: %v", r)
		}
	}()

	version, r := l.nvmllib.SystemGetDriverVersion()
	if r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to determine driver version: %v", r)
	}

	return (*nvcdilib)(l).newDriverVersionDiscoverer(version)
}

func (l *nvcdilib) newDriverVersionDiscoverer(version string) (discover.Discover, error) {
	libraries, err := NewDriverLibraryDiscoverer(l.logger, l.driver, l.nvidiaCDIHookPath, l.ldconfigPath, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for driver libraries: %v", err)
	}

	ipcs, err := discover.NewIPCDiscoverer(l.logger, l.driver.Root, l.optInFeatures["include-persistenced-socket"])
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for IPC sockets: %v", err)
	}

	firmwares, err := l.newDriverFirmwareDiscoverer(l.logger, l.driver.Root, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for GSP firmware: %v", err)
	}

	binaries := NewDriverBinariesDiscoverer(l.logger, l.driver.Root)

	d := discover.Merge(
		libraries,
		ipcs,
		firmwares,
		binaries,
	)

	return d, nil
}

// NewDriverLibraryDiscoverer creates a discoverer for the libraries associated with the specified driver version.
func NewDriverLibraryDiscoverer(logger logger.Interface, driver *root.Driver, nvidiaCDIHookPath, ldconfigPath, version string) (discover.Discover, error) {
	libraryPaths, err := getVersionLibs(logger, driver, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries for driver version: %v", err)
	}

	libraries := discover.NewMounts(
		logger,
		lookup.NewFileLocator(
			lookup.WithLogger(logger),
			lookup.WithRoot(driver.Root),
		),
		driver.Root,
		libraryPaths,
	)

	updateLDCache, _ := discover.NewLDCacheUpdateHook(logger, libraries, nvidiaCDIHookPath, ldconfigPath)

	d := discover.Merge(
		discover.WithDriverDotSoSymlinks(
			libraries,
			version,
			nvidiaCDIHookPath,
		),
		updateLDCache,
	)

	return d, nil
}

// NewDriverBinariesDiscoverer creates a discoverer for GSP firmware associated with the GPU driver.
func NewDriverBinariesDiscoverer(logger logger.Interface, driverRoot string) discover.Discover {
	return discover.NewMounts(
		logger,
		lookup.NewExecutableLocator(logger, driverRoot),
		driverRoot,
		[]string{
			"nvidia-smi",              /* System management interface */
			"nvidia-debugdump",        /* GPU coredump utility */
			"nvidia-persistenced",     /* Persistence mode utility */
			"nvidia-cuda-mps-control", /* Multi process service CLI */
			"nvidia-cuda-mps-server",  /* Multi process service server */
		},
	)
}

// getVersionLibs checks the LDCache for libraries ending in the specified driver version.
// Although the ldcache at the specified driverRoot is queried, the paths are returned relative to this driverRoot.
// This allows the standard mount location logic to be used for resolving the mounts.
func getVersionLibs(logger logger.Interface, driver *root.Driver, version string) ([]string, error) {
	logger.Infof("Using driver version %v", version)

	libCudaPaths, err := cuda.New(
		driver.Libraries(),
	).Locate("." + version)
	if err != nil {
		return nil, fmt.Errorf("failed to locate libcuda.so.%v: %v", version, err)
	}
	libRoot := filepath.Dir(libCudaPaths[0])

	libraries := lookup.NewFileLocator(
		lookup.WithLogger(logger),
		lookup.WithSearchPaths(
			libRoot,
			filepath.Join(libRoot, "vdpau"),
		),
		lookup.WithOptional(true),
	)

	libs, err := libraries.Locate("*.so." + version)
	if err != nil {
		return nil, fmt.Errorf("failed to locate libraries for driver version %v: %v", version, err)
	}

	if driver.Root == "/" || driver.Root == "" {
		return libs, nil
	}

	var relative []string
	for _, l := range libs {
		relative = append(relative, strings.TrimPrefix(l, driver.Root))
	}

	return relative, nil
}
