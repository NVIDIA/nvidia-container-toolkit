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

package engine

const (
	// SaveToSTDOUT is used to write the specified config to stdout instead of
	// to a file on disk.
	SaveToSTDOUT = ""
	// UpdateActionSet is used as an argument to UpdateDefaultRuntime
	// when setting a runtime handler as the default in the config
	UpdateActionSet = "set"
	// UpdateActionUnset is used as an argument to UpdateDefaultRuntime
	// when unsetting a runtime handler as the default in the config
	UpdateActionUnset = "unset"
)

// Interface defines the API for a runtime config updater.
type Interface interface {
	AddRuntime(string, string, bool) error
	DefaultRuntime() string
	EnableCDI()
	GetRuntimeConfig(string) (RuntimeConfig, error)
	RemoveRuntime(string) error
	UpdateDefaultRuntime(string, string) error
	Save(string) (int64, error)
	String() string
}

// RuntimeConfig defines the interface to query container runtime handler configuration
type RuntimeConfig interface {
	GetBinaryPath() string
}
