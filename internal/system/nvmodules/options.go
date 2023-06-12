/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package nvmodules

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

// Option is a function that sets an option on the Interface struct.
type Option func(*Interface)

// WithDryRun sets the dry run option for the Interface struct.
func WithDryRun(dryRun bool) Option {
	return func(i *Interface) {
		i.dryRun = dryRun
	}
}

// WithLogger sets the logger for the Interface struct.
func WithLogger(logger logger.Interface) Option {
	return func(i *Interface) {
		i.logger = logger
	}
}

// WithRoot sets the root directory for the NVIDIA device nodes.
func WithRoot(root string) Option {
	return func(i *Interface) {
		i.root = root
	}
}
