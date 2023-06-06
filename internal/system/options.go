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

package system

import "github.com/sirupsen/logrus"

// Option is a functional option for the system command
type Option func(*Interface)

// WithLogger sets the logger for the system command
func WithLogger(logger *logrus.Logger) Option {
	return func(i *Interface) {
		i.logger = logger
	}
}

// WithDryRun sets the dry run flag
func WithDryRun(dryRun bool) Option {
	return func(i *Interface) {
		i.dryRun = dryRun
	}
}

// WithLoadKernelModules sets the load kernel modules flag
func WithLoadKernelModules(loadKernelModules bool) Option {
	return func(i *Interface) {
		i.loadKernelModules = loadKernelModules
	}
}
