/**
# Copyright (c) 2020-2021, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"github.com/pelletier/go-toml"
)

// UpdateReverter defines the interface for applying and reverting configurations
type UpdateReverter interface {
	Update(o *options) error
	Revert(o *options) error
}

type config struct {
	*toml.Tree
	version int64
	cri     string
}

// update adds the specified runtime class to the the containerd config.
// if set-as default is specified, the runtime class is also set as the
// default runtime.
func (config *config) update(runtimeClass string, runtimeType string, runtimeBinary string, setAsDefault bool) {
	config.Set("version", config.version)

	runcPath := config.runcPath()
	runtimeClassPath := config.runtimeClassPath(runtimeClass)

	switch runc := config.GetPath(runcPath).(type) {
	case *toml.Tree:
		runc, _ = toml.Load(runc.String())
		config.SetPath(runtimeClassPath, runc)
	}

	config.initRuntime(runtimeClassPath, runtimeType, "BinaryName", runtimeBinary)
	if config.version == 1 {
		config.initRuntime(runtimeClassPath, runtimeType, "Runtime", runtimeBinary)
	}

	if setAsDefault {
		defaultRuntimeNamePath := config.defaultRuntimeNamePath()
		config.SetPath(defaultRuntimeNamePath, runtimeClass)
	}
}

// revert removes the configuration applied in an update call.
func (config *config) revert(runtimeClass string) {
	runtimeClassPath := config.runtimeClassPath(runtimeClass)
	defaultRuntimeNamePath := config.defaultRuntimeNamePath()

	config.DeletePath(runtimeClassPath)
	if runtime, ok := config.GetPath(defaultRuntimeNamePath).(string); ok {
		if runtimeClass == runtime {
			config.DeletePath(defaultRuntimeNamePath)
		}
	}

	for i := 0; i < len(runtimeClassPath); i++ {
		if runtimes, ok := config.GetPath(runtimeClassPath[:len(runtimeClassPath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				config.DeletePath(runtimeClassPath[:len(runtimeClassPath)-i])
			}
		}
	}

	if len(config.Keys()) == 1 && config.Keys()[0] == "version" {
		config.Delete("version")
	}
}

// initRuntime creates a runtime config if it does not exist and ensures that the
// runtimes binary path is specified.
func (config *config) initRuntime(path []string, runtimeType string, binaryKey string, binary string) {
	if config.GetPath(path) == nil {
		config.SetPath(append(path, "runtime_type"), runtimeType)
		config.SetPath(append(path, "runtime_root"), "")
		config.SetPath(append(path, "runtime_engine"), "")
		config.SetPath(append(path, "privileged_without_host_devices"), false)
	}

	binaryPath := append(path, "options", binaryKey)
	config.SetPath(binaryPath, binary)
}

func (config config) runcPath() []string {
	return config.runtimeClassPath("runc")
}

func (config config) runtimeClassPath(runtimeClass string) []string {
	return append(config.containerdPath(), "runtimes", runtimeClass)
}

func (config config) defaultRuntimeNamePath() []string {
	return append(config.containerdPath(), "default_runtime_name")
}

func (config config) containerdPath() []string {
	return []string{"plugins", config.cri, "containerd"}
}
