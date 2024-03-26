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

package info

// Interface provides the API to the info package
type Interface interface {
	Resolver
	Properties
}

// Resolver defines a function to resolve a mode.
type Resolver interface {
	Resolve(string) string
}

// Properties provides a set of functions to query capabilities of the system.
//
//go:generate moq -rm -stub -out properties_mock.go . Properties
type Properties interface {
	HasDXCore() (bool, string)
	HasNvml() (bool, string)
	IsTegraSystem() (bool, string)
	UsesNVGPUModule() (bool, string)
}
