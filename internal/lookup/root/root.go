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
	"path/filepath"
	"strings"
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/cuda"
)

// Driver represents a filesystem in which a set of drivers or devices is defined.
type Driver struct {
	sync.Mutex
	logger logger.Interface
	// Root represents the root from the perspective of the driver libraries and binaries.
	Root string
	// librarySearchPaths specifies explicit search paths for discovering libraries.
	librarySearchPaths []string
	// version stores the driver version. This can be specified at construction or cached on subsequent calls.
	version string
	// libraryRoot stores the absolute path where the driver libraries (libcuda.so.<VERSION>) can be found.
	libraryRoot string
}

// New creates a new Driver root at the specified path.
// TODO: Use functional options here.
func New(logger logger.Interface, path string, librarySearchPaths []string, version string) *Driver {
	return &Driver{
		logger:             logger,
		Root:               path,
		librarySearchPaths: normalizeSearchPaths(librarySearchPaths...),
		version:            version,
	}
}

// Drivers returns a Locator for driver libraries.
func (r *Driver) Libraries() lookup.Locator {
	return lookup.NewLibraryLocator(
		lookup.WithLogger(r.logger),
		lookup.WithRoot(r.Root),
		lookup.WithSearchPaths(r.librarySearchPaths...),
	)
}

// Version returns the driver version as a string.
func (r *Driver) Version() (string, error) {
	r.Lock()
	defer r.Unlock()
	if r.version != "" {
		return r.version, nil
	}

	libCudaPaths, err := cuda.New(
		r.Libraries(),
	).Locate(".*.*")
	if err != nil {
		return "", fmt.Errorf("failed to locate libcuda.so: %v", err)
	}
	libcudaPath := libCudaPaths[0]

	version := strings.TrimPrefix(filepath.Base(libcudaPath), "libcuda.so.")
	if version == "" {
		return "", fmt.Errorf("failed to determine libcuda.so version from path: %q", libcudaPath)
	}

	r.version = version
	return r.version, nil
}

// LibraryRoot returns the folder in which the driver libraries can be found.
func (r *Driver) LibraryRoot() (string, error) {
	r.Lock()
	defer r.Unlock()
	if r.libraryRoot != "" {
		return r.libraryRoot, nil
	}

	libCudaPaths, err := cuda.New(
		r.Libraries(),
	).Locate(".*.*")
	if err != nil {
		return "", fmt.Errorf("failed to locate libcuda.so: %v", err)
	}

	r.libraryRoot = filepath.Dir(libCudaPaths[0])
	return r.libraryRoot, nil
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
