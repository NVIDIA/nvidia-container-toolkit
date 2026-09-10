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
// library locator combines the unique matches from the following sources, in
// precedence order:
//   - a set of predefined search paths
//   - the 64-bit entries of the ldcache
//   - the 32-bit entries of the ldcache
//
// A 32-bit library is typically in a directory that is not in the predefined
// search paths, so the ldcache is consulted even if one of these already
// provided a match. If 32-bit libraries are excluded
// (WithCompat32Libraries(false)), the first source with a match is used
// instead.
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
	if !f.compat32 {
		return First(
			NewSymlinkLocator(opts...),
			f.newLdcacheLocator(),
		)
	}

	return AsUnique(Merge(
		NewSymlinkLocator(opts...),
		f.newLdcacheLocator(),
	))
}
