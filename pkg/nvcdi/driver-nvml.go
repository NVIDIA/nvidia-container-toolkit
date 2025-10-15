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
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// NewDriverDiscoverer creates a discoverer for the libraries and binaries associated with a driver installation.
// The supplied NVML Library is used to query the expected driver version.
func (l *nvmllib) NewDriverDiscoverer() (discover.Discover, error) {
	return (*nvcdilib)(l).newDriverVersionDiscoverer()
}

func (l *nvcdilib) newDriverVersionDiscoverer() (discover.Discover, error) {
	version, err := l.driver.Version()
	if err != nil || version == "" || version == "*.*" {
		return nil, fmt.Errorf("failed to determine driver version (%q): %w", version, err)
	}

	libcudasoParentDirPath, err := l.driver.GetLibcudaParentDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get libcuda.so parent path: %w", err)
	}

	libraries, err := l.NewDriverLibraryDiscoverer(version, libcudasoParentDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for driver libraries: %v", err)
	}

	ipcs, err := discover.NewIPCDiscoverer(l.logger, l.driver.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for IPC sockets: %v", err)
	}

	firmwares, err := NewDriverFirmwareDiscoverer(l.logger, l.driver.Root, version)
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
func (l *nvcdilib) NewDriverLibraryDiscoverer(version string, libcudaSoParentDirPath string) (discover.Discover, error) {
	versionSuffixLibraryMounts, err := l.getVersionSuffixDriverLibraryMounts(version)
	if err != nil {
		return nil, err
	}
	explicitLibraryMounts, err := l.getExplicitDriverLibraryMounts()
	if err != nil {
		return nil, err
	}

	libraries := discover.Merge(
		versionSuffixLibraryMounts,
		explicitLibraryMounts,
	)

	var discoverers []discover.Discover

	driverDotSoSymlinksDiscoverer := discover.WithDriverDotSoSymlinks(
		l.logger,
		libraries,
		// Since we don't only match version suffixes, we now need to match on wildcards.
		"",
		l.hookCreator,
	)
	discoverers = append(discoverers, driverDotSoSymlinksDiscoverer)

	cudaCompatLibHookDiscoverer := discover.NewCUDACompatHookDiscoverer(l.logger, l.hookCreator, version)
	discoverers = append(discoverers, cudaCompatLibHookDiscoverer)

	updateLDCache, _ := discover.NewLDCacheUpdateHook(l.logger, libraries, l.hookCreator, l.ldconfigPath)
	discoverers = append(discoverers, updateLDCache)

	disableDeviceNodeModification := l.hookCreator.Create(DisableDeviceNodeModificationHook)
	discoverers = append(discoverers, disableDeviceNodeModification)

	libCudaSoParentDirectoryPath, err := l.driver.GetLibcudaParentDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get libcuda.so parent directory path: %w", err)
	}
	environmentVariable := &discover.EnvVar{
		Name:  "NVIDIA_CTK_LIBCUDA_DIR",
		Value: libCudaSoParentDirectoryPath,
	}
	discoverers = append(discoverers, environmentVariable)

	d := discover.Merge(discoverers...)

	return d, nil
}

func (l *nvcdilib) getVersionSuffixDriverLibraryMounts(version string) (discover.Discover, error) {
	versionSuffixLibraryPaths, err := getVersionLibs(l.logger, l.driver, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries for driver version: %v", err)
	}

	mounts := discover.NewMounts(
		l.logger,
		lookup.NewFileLocator(
			lookup.WithLogger(l.logger),
			lookup.WithRoot(l.driver.Root),
		),
		l.driver.Root,
		versionSuffixLibraryPaths,
	)

	return mounts, nil
}

func (l *nvcdilib) getExplicitDriverLibraryMounts() (discover.Discover, error) {
	if !l.featureFlags[FeatureEnableExplicitDriverLibraries] {
		return nil, nil
	}

	// List of explicit libraries to locate
	// TODO(ArangoGutierrez): we should load the version of the libraries from
	// the sandboxutils-filelist or have a way to allow users to specify the
	// libraries to mount from the config file.
	explicitLibraries := []string{
		"libEGL.so",
		"libGL.so",
		"libGLESv1_CM.so",
		"libGLESv2.so",
		"libGLX.so",
		"libGLdispatch.so",
		"libOpenCL.so",
		"libOpenGL.so",
		"libnvidia-api.so",
		"libnvidia-egl-xcb.so",
		"libnvidia-egl-xlib.so",
	}

	driverLibraryLocator, err := l.driver.DriverLibraryLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to get driver library locator: %w", err)
	}
	mounts := discover.NewMounts(
		l.logger,
		driverLibraryLocator,
		l.driver.Root,
		explicitLibraries,
	)

	return mounts, nil

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

// NewDriverFirmwareDiscoverer creates a discoverer for GSP firmware associated with the specified driver version.
func NewDriverFirmwareDiscoverer(logger logger.Interface, driverRoot string, version string) (discover.Discover, error) {
	gspFirmwareSearchPaths, err := getFirmwareSearchPaths(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware search paths: %v", err)
	}
	gspFirmwarePaths := filepath.Join("nvidia", version, "gsp*.bin")
	return discover.NewMounts(
		logger,
		lookup.NewFileLocator(
			lookup.WithLogger(logger),
			lookup.WithRoot(driverRoot),
			lookup.WithSearchPaths(gspFirmwareSearchPaths...),
		),
		driverRoot,
		[]string{gspFirmwarePaths},
	), nil
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

	libraries, err := driver.DriverLibraryLocator("vdpau")
	if err != nil {
		return nil, fmt.Errorf("failed to get driver library locator: %w", err)
	}

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
