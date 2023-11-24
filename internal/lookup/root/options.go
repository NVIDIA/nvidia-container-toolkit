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

package root

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

type Option func(*Driver)

func WithLogger(logger logger.Interface) Option {
	return func(d *Driver) {
		d.logger = logger
	}
}

func WithDriverRoot(root string) Option {
	return func(d *Driver) {
		d.Root = root
	}
}

func WithLibrarySearchPaths(paths ...string) Option {
	return func(d *Driver) {
		d.librarySearchPaths = paths
	}
}

func WithConfigSearchPaths(paths ...string) Option {
	return func(d *Driver) {
		d.configSearchPaths = paths
	}
}
