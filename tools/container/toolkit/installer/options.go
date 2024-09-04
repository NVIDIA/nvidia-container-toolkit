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

package installer

type Option func(*toolkitInstaller)

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

func WithPackageType(packageType string) Option {
	return func(ti *toolkitInstaller) {
		ti.packageType = packageType
	}
}

func WithHostRoot(hostRoot string) Option {
	return func(ti *toolkitInstaller) {
		ti.hostRoot = hostRoot
	}
}
