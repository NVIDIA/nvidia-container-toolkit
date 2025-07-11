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
func NewLibraryLocator(opts ...Option) Locator {
	b := newBuilder(opts...)

	// If search paths are already specified, we return a locator for the specified search paths.
	if len(b.searchPaths) > 0 {
		return NewSymlinkLocator(
			WithLogger(b.logger),
			WithSearchPaths(b.searchPaths...),
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
	// We construct a symlink locator for expected library locations.
	symlinkLocator := NewSymlinkLocator(opts...)

	l := First(
		symlinkLocator,
		NewLdcacheLocator(opts...),
	)
	return l
}
