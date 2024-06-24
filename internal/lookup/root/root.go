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
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

// Driver represents a filesystem in which a set of drivers or devices is defined.
type Driver struct {
	logger logger.Interface
	// Root represents the root from the perspective of the driver libraries and binaries.
	Root string
	// librarySearchPaths specifies explicit search paths for discovering libraries.
	librarySearchPaths []string
	// configSearchPaths specified explicit search paths for discovering driver config files.
	configSearchPaths []string
}

// New creates a new Driver root using the specified options.
func New(opts ...Option) *Driver {
	d := &Driver{}
	for _, opt := range opts {
		opt(d)
	}
	if d.logger == nil {
		d.logger = logger.New()
	}
	return d
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
