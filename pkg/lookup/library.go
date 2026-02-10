/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package lookup

// NewLibraryLocator creates a library locator using the specified options.
// If search paths (WithSearchPaths(path1, path2, ...)) are explicitly specified
// a library locator using these as absolute paths are used. Otherwise the
// library is constructed using the following ordering, returning the first
// successful result:
//   - attempt to locate the library / pattern using dlopen
//   - attempt to locate the library from a set of predefined search paths.
//   - attempt to locate the library from the ldcache.
func NewLibraryLocator(opts ...Option) Locator {
	f := NewFactory(opts...)

	// If search paths are already specified, we return a locator for the specified search paths.
	if len(f.searchPaths) > 0 {
		return NewSymlinkLocator(
			WithLogger(f.logger),
			WithSearchPaths(f.searchPaths...),
			WithRoot("/"),
		)
	}
	opts = append(opts,
		WithSearchPaths([]string{
			"/",
			"/usr/lib64",
			"/usr/lib/x86_64-linux-gnu",
			"/usr/lib/aarch64-linux-gnu",
			"/usr/lib/x86_64-linux-gnu/nvidia/current",
			"/usr/lib/aarch64-linux-gnu/nvidia/current",
			"/lib64",
			"/lib/x86_64-linux-gnu",
			"/lib/aarch64-linux-gnu",
			"/lib/x86_64-linux-gnu/nvidia/current",
			"/lib/aarch64-linux-gnu/nvidia/current",
		}...),
	)
	l := First(
		f.newDlopenLocator(),
		NewSymlinkLocator(opts...),
		f.newLdcacheLocator(),
	)
	return l
}
