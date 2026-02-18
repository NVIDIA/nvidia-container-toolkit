/**
# Copyright 2023 NVIDIA CORPORATION
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

package root

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

// Driver represents a filesystem in which a set of drivers or devices is defined.
type Driver struct {
	sync.Mutex
	logger logger.Interface
	// Root represents the root from the perspective of the driver libraries and binaries.
	Root string
	// librarySearchPaths specifies explicit search paths for discovering libraries.
	librarySearchPaths []string
	// configSearchPaths specified explicit search paths for discovering driver config files.
	configSearchPaths []string

	// version caches the driver version.
	version string
	// driverLibDirectory caches the path to parent of the driver libraries
	driverLibDirectory string
}

// New creates a new Driver root using the specified options.
func New(opts ...Option) *Driver {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = logger.New()
	}

	var driverVersion string
	if o.versioner != nil {
		version, err := o.versioner.Version()
		if err != nil {
			o.logger.Warningf("Could not determine driver version: %v", err)
		}
		driverVersion = version
	}

	d := &Driver{
		logger:             o.logger,
		Root:               o.Root,
		librarySearchPaths: o.librarySearchPaths,
		configSearchPaths:  o.configSearchPaths,
		version:            driverVersion,
		driverLibDirectory: "",
	}

	return d
}

// Version returns the cached driver version if possible.
// If this has not yet been initialised, the version is first updated and then returned.
func (r *Driver) Version() (string, error) {
	r.Lock()
	defer r.Unlock()

	if r.version == "" {
		if err := r.updateInfo(); err != nil {
			return "", err
		}
	}

	return r.version, nil
}

// GetDriverLibDirectory returns the cached directory where the driver libs are
// found if possible.
// If this has not yet been initialized, the path is first detected and then returned.
func (r *Driver) GetDriverLibDirectory() (string, error) {
	r.Lock()
	defer r.Unlock()

	if r.driverLibDirectory == "" {
		if err := r.updateInfo(); err != nil {
			return "", err
		}
	}

	return r.driverLibDirectory, nil
}

func (r *Driver) DriverLibraryLocator(additionalDirs ...string) (lookup.Locator, error) {
	libcudasoParentDirPath, err := r.GetDriverLibDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get libcuda.so parent directory: %w", err)
	}

	searchPaths := []string{libcudasoParentDirPath}
	for _, dir := range additionalDirs {
		if strings.HasPrefix(dir, "/") {
			searchPaths = append(searchPaths, dir)
		} else {
			searchPaths = append(searchPaths, filepath.Join(libcudasoParentDirPath, dir))
		}
	}

	l := lookup.AsOptional(
		lookup.NewSymlinkLocator(
			lookup.WithRoot(r.Root),
			lookup.WithLogger(r.logger),
			lookup.WithSearchPaths(
				searchPaths...,
			),
		),
	)
	return l, nil
}

func (r *Driver) updateInfo() error {
	driverLibPath, version, err := r.inferVersion()
	if err != nil {
		return err
	}
	if r.version != "" && r.version != version {
		return fmt.Errorf("unexpected version detected: %v != %v", r.version, version)
	}

	r.version = version
	r.driverLibDirectory = r.RelativeToRoot(filepath.Dir(driverLibPath))

	return nil
}

// inferVersion attempts to infer the driver version from the libcuda.so or
// libnvidia-ml.so driver library suffixes.
func (r *Driver) inferVersion() (string, string, error) {
	versionSuffix := r.version
	if versionSuffix == "" {
		versionSuffix = "*.*"
	}

	var errs error
	for _, driverLib := range []string{"libcuda.so.", "libnvidia-ml.so."} {
		driverLibPaths, err := r.Libraries().Locate(driverLib + versionSuffix)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to locate libcuda.so: %w", err))
			continue
		}
		driverLibPath := driverLibPaths[0]
		version := strings.TrimPrefix(filepath.Base(driverLibPath), driverLib)
		if version == "" {
			errs = errors.Join(errs, fmt.Errorf("failed to extract version from path %v", driverLibPath))
			continue
		}
		return driverLibPath, version, nil
	}

	return "", "", errs
}

// RelativeToRoot returns the specified path relative to the driver root.
func (r *Driver) RelativeToRoot(path string) string {
	if r.Root == "" || r.Root == "/" {
		return path
	}
	if !filepath.IsAbs(path) {
		return path
	}

	return strings.TrimPrefix(path, r.Root)
}

// Files returns a Locator for arbitrary driver files.
func (r *Driver) Files(opts ...lookup.Option) lookup.Locator {
	return lookup.NewFileLocator(
		append(opts,
			lookup.WithLogger(r.logger),
			lookup.WithRoot(r.Root),
		)...,
	)
}

// Libraries returns a Locator for driver libraries.
func (r *Driver) Libraries() lookup.Locator {
	return lookup.NewLibraryLocator(
		lookup.WithLogger(r.logger),
		lookup.WithRoot(r.Root),
		lookup.WithSearchPaths(r.librarySearchPaths...),
	)
}

// Configs returns a locator for driver configs.
// If configSearchPaths is specified, these paths are used as absolute paths,
// otherwise, /etc and /usr/share are searched.
func (r *Driver) Configs() lookup.Locator {
	return lookup.NewFileLocator(r.configSearchOptions()...)
}

func (r *Driver) configSearchOptions() []lookup.Option {
	if len(r.configSearchPaths) > 0 {
		return []lookup.Option{
			lookup.WithLogger(r.logger),
			lookup.WithRoot("/"),
			lookup.WithSearchPaths(r.configSearchPaths...),
		}
	}
	searchPaths := []string{"/etc"}
	searchPaths = append(searchPaths, xdgDataDirs()...)
	return []lookup.Option{
		lookup.WithLogger(r.logger),
		lookup.WithRoot(r.Root),
		lookup.WithSearchPaths(searchPaths...),
	}
}

// xdgDataDirs finds the paths as specified in the environment variable XDG_DATA_DIRS.
// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html.
func xdgDataDirs() []string {
	if dirs, exists := os.LookupEnv("XDG_DATA_DIRS"); exists && dirs != "" {
		return lookup.NormalizePaths(dirs)
	}

	return []string{"/usr/local/share", "/usr/share"}
}
