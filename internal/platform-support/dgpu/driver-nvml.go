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

package dgpu

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/cuda"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// newNvmlDriverDiscoverer constructs a discoverer from the specified NVML library.
func (o *options) newNvmlDriverDiscoverer() (discover.Discover, error) {
	libraries, err := o.newNvmlDriverLibraryDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for driver libraries: %v", err)
	}

	firmwares, err := o.newNvmlDriverFirmwareDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for GSP firmware: %v", err)
	}

	binaries := o.newNvmlDriverBinariesDiscoverer()

	d := discover.Merge(
		libraries,
		firmwares,
		binaries,
	)

	return d, nil
}

// newNvmlDriverLibraryDiscoverer creates a discoverer for the libraries associated with the specified driver version.
func (o *options) newNvmlDriverLibraryDiscoverer() (discover.Discover, error) {
	libraryPaths, err := getVersionLibs(o.logger, o.driver, o.version)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries for driver version: %v", err)
	}

	libraries := discover.NewMounts(
		o.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(o.logger),
			lookup.WithRoot(o.driver.Root),
		),
		o.driver.Root,
		libraryPaths,
	)

	d := discover.WithDriverDotSoSymlinks(
		libraries,
		o.version,
		o.nvidiaCDIHookPath,
	)

	return d, nil
}

func getUTSRelease() (string, error) {
	utsname := &unix.Utsname{}
	if err := unix.Uname(utsname); err != nil {
		return "", err
	}
	return unix.ByteSliceToString(utsname.Release[:]), nil
}

func getFirmwareSearchPaths(logger logger.Interface) ([]string, error) {

	var firmwarePaths []string
	if p := getCustomFirmwareClassPath(logger); p != "" {
		logger.Debugf("using custom firmware class path: %s", p)
		firmwarePaths = append(firmwarePaths, p)
	}

	utsRelease, err := getUTSRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to get UTS_RELEASE: %v", err)
	}

	standardPaths := []string{
		filepath.Join("/lib/firmware/updates/", utsRelease),
		"/lib/firmware/updates/",
		filepath.Join("/lib/firmware/", utsRelease),
		"/lib/firmware/",
	}

	return append(firmwarePaths, standardPaths...), nil
}

// getCustomFirmwareClassPath returns the custom firmware class path if it exists.
func getCustomFirmwareClassPath(logger logger.Interface) string {
	customFirmwareClassPath, err := os.ReadFile("/sys/module/firmware_class/parameters/path")
	if err != nil {
		logger.Warningf("failed to get custom firmware class path: %v", err)
		return ""
	}

	return strings.TrimSpace(string(customFirmwareClassPath))
}

// newNvmlDriverFirmwareDiscoverer creates a discoverer for GSP firmware associated with the specified driver version.
func (o *options) newNvmlDriverFirmwareDiscoverer() (discover.Discover, error) {
	gspFirmwareSearchPaths, err := getFirmwareSearchPaths(o.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware search paths: %v", err)
	}
	gspFirmwarePaths := filepath.Join("nvidia", o.version, "gsp*.bin")
	return discover.NewMounts(
		o.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(o.logger),
			lookup.WithRoot(o.driver.Root),
			lookup.WithSearchPaths(gspFirmwareSearchPaths...),
		),
		o.driver.Root,
		[]string{gspFirmwarePaths},
	), nil
}

// newNvmlDriverBinariesDiscoverer creates a discoverer for binaries associated with the specified driver version.
func (o *options) newNvmlDriverBinariesDiscoverer() discover.Discover {
	return discover.NewMounts(
		o.logger,
		lookup.NewExecutableLocator(o.logger, o.driver.Root),
		o.driver.Root,
		[]string{
			"nvidia-smi",              /* System management interface */
			"nvidia-debugdump",        /* GPU coredump utility */
			"nvidia-persistenced",     /* Persistence mode utility */
			"nvidia-cuda-mps-control", /* Multi process service CLI */
			"nvidia-cuda-mps-server",  /* Multi process service server */
			"nvidia-imex",             /* NVIDIA IMEX Daemon */
			"nvidia-imex-ctl",         /* NVIDIA IMEX control */
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
