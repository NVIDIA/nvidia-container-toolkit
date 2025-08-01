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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
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
	// libcudasoPath caches the path to libcuda.so.VERSION.
	libcudasoPath string
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
		libcudasoPath:      "",
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

// GetLibcudaParentDir returns the cached libcuda.so path if possible.
// If this has not yet been initialized, the path is first detected and then returned.
func (r *Driver) GetLibcudasoPath() (string, error) {
	r.Lock()
	defer r.Unlock()

	if r.libcudasoPath == "" {
		if err := r.updateInfo(); err != nil {
			return "", err
		}
	}

	return r.libcudasoPath, nil
}

func (r *Driver) GetLibcudaParentDir() (string, error) {
	libcudasoPath, err := r.GetLibcudasoPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(libcudasoPath), nil
}

func (r *Driver) DriverLibraryLocator(additionalDirs ...string) (lookup.Locator, error) {
	libcudasoParentDirPath, err := r.GetLibcudaParentDir()
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

	l := lookup.NewFileLocator(
		lookup.WithRoot(r.Root),
		lookup.WithLogger(r.logger),
		lookup.WithSearchPaths(
			searchPaths...,
		),
		lookup.WithOptional(true),
	)
	return l, nil
}

func (r *Driver) updateInfo() error {
	versionSuffix := r.version
	if versionSuffix == "" {
		versionSuffix = "*.*"
	}

	libCudaPaths, err := r.Libraries().Locate("libcuda.so." + versionSuffix)
	if err != nil {
		return fmt.Errorf("failed to locate libcuda.so: %w", err)
	}
	libcudaPath := libCudaPaths[0]

	version := strings.TrimPrefix(filepath.Base(libcudaPath), "libcuda.so.")
	if version == "" {
		return fmt.Errorf("failed to extract version from path %v", libcudaPath)
	}

	if r.version != "" && r.version != version {
		return fmt.Errorf("unexpected version detected: %v != %v", r.version, version)
	}
	r.version = version
	r.libcudasoPath = r.RelativeToRoot(libcudaPath)
	return nil
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
		lookup.WithSearchPaths(normalizeSearchPaths(r.librarySearchPaths...)...),
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
			lookup.WithSearchPaths(normalizeSearchPaths(r.configSearchPaths...)...),
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

// normalizeSearchPaths takes a list of paths and normalized these.
// Each of the elements in the list is expanded if it is a path list and the
// resultant list is returned.
// This allows, for example, for the contents of `PATH` or `LD_LIBRARY_PATH` to
// be passed as a search path directly.
func normalizeSearchPaths(paths ...string) []string {
	var normalized []string
	for _, path := range paths {
		normalized = append(normalized, filepath.SplitList(path)...)
	}
	return normalized
}

// xdgDataDirs finds the paths as specified in the environment variable XDG_DATA_DIRS.
// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html.
func xdgDataDirs() []string {
	if dirs, exists := os.LookupEnv("XDG_DATA_DIRS"); exists && dirs != "" {
		return normalizeSearchPaths(dirs)
	}

	return []string{"/usr/local/share", "/usr/share"}
}
