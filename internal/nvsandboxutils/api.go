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

package nvsandboxutils

// libraryOptions hold the parameters than can be set by a LibraryOption
type libraryOptions struct {
	path  string
	flags int
}

// LibraryOption represents a functional option to configure the underlying nvsandboxutils library
type LibraryOption func(*libraryOptions)

// WithLibraryPath provides an option to set the library name to be used by the nvsandboxutils library.
func WithLibraryPath(path string) LibraryOption {
	return func(o *libraryOptions) {
		o.path = path
	}
}

// SetLibraryOptions applies the specified options to the nvsandboxutils library.
// If this is called when a library is already loaded, an error is raised.
func SetLibraryOptions(opts ...LibraryOption) error {
	libnvsandboxutils.Lock()
	defer libnvsandboxutils.Unlock()
	if libnvsandboxutils.refcount != 0 {
		return errLibraryAlreadyLoaded
	}
	libnvsandboxutils.init(opts...)
	return nil
}
