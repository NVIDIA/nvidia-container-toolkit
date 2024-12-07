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

package toolkit

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

// An Option provides a mechanism to configure an Installer.
type Option func(*Installer)

func WithLogger(logger logger.Interface) Option {
	return func(i *Installer) {
		i.logger = logger
	}
}

func WithToolkitRoot(toolkitRoot string) Option {
	return func(i *Installer) {
		i.toolkitRoot = toolkitRoot
	}
}

func WithSourceRoot(sourceRoot string) Option {
	return func(i *Installer) {
		i.sourceRoot = sourceRoot
	}
}
