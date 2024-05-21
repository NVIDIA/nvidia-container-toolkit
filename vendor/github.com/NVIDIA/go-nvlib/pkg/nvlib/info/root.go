/**
# Copyright 2024 NVIDIA CORPORATION
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

package info

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

// root represents a directory on the filesystem relative to which libraries
// such as the NVIDIA driver libraries can be found.
type root string

func (r root) join(parts ...string) string {
	return filepath.Join(append([]string{string(r)}, parts...)...)
}

// assertHasLibrary returns an error if the specified library cannot be loaded.
func (r root) assertHasLibrary(libraryName string) error {
	const (
		libraryLoadFlags = dl.RTLD_LAZY
	)
	lib := dl.New(r.tryResolveLibrary(libraryName), libraryLoadFlags)
	if err := lib.Open(); err != nil {
		return err
	}
	defer lib.Close()

	return nil
}

// tryResolveLibrary attempts to locate the specified library in the root.
// If the root is not specified, is "/", or the library cannot be found in the
// set of predefined paths, the input is returned as is.
func (r root) tryResolveLibrary(libraryName string) string {
	if r == "" || r == "/" {
		return libraryName
	}

	librarySearchPaths := []string{
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
		"/lib64",
		"/lib/x86_64-linux-gnu",
		"/lib/aarch64-linux-gnu",
	}

	for _, d := range librarySearchPaths {
		l := r.join(d, libraryName)
		resolved, err := resolveLink(l)
		if err != nil {
			continue
		}
		return resolved
	}

	return libraryName
}

// resolveLink finds the target of a symlink or the file itself in the
// case of a regular file.
// This is equivalent to running `readlink -f ${l}`.
func resolveLink(l string) (string, error) {
	resolved, err := filepath.EvalSymlinks(l)
	if err != nil {
		return "", fmt.Errorf("error resolving link '%v': %w", l, err)
	}
	return resolved, nil
}
