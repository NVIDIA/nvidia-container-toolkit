/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package installer

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

type Option func(*toolkitInstaller)

func WithLogger(logger logger.Interface) Option {
	return func(ti *toolkitInstaller) {
		ti.logger = logger
	}
}

func WithArtifactRoot(artifactRoot *artifactRoot) Option {
	return func(ti *toolkitInstaller) {
		ti.artifactRoot = artifactRoot
	}
}

func WithIgnoreErrors(ignoreErrors bool) Option {
	return func(ti *toolkitInstaller) {
		ti.ignoreErrors = ignoreErrors
	}
}

// WithSourceRoot sets the root directory for locating artifacts to be installed.
func WithSourceRoot(sourceRoot string) Option {
	return func(ti *toolkitInstaller) {
		ti.sourceRoot = sourceRoot
	}
}
