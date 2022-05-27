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
	"path"

	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
)

// configV1 represents a V1 containerd config
type configV1 struct {
	config
}

func newConfigV1(cfg *toml.Tree) UpdateReverter {
	c := configV1{
		config: config{
			Tree:    cfg,
			version: 1,
			cri:     "cri",
		},
	}

	return &c
}

// Update performs an update specific to v1 of the containerd config
func (config *configV1) Update(o *options) error {

	// For v1 config, the `default_runtime_name` setting is only supported
	// for containerd version at least v1.3
	supportsDefaultRuntimeName := !o.useLegacyConfig

	defaultRuntime := o.getDefaultRuntime()

	for runtimeClass, runtimeBinary := range o.getRuntimeBinaries() {
		isDefaultRuntime := runtimeClass == defaultRuntime
		config.update(runtimeClass, o.runtimeType, runtimeBinary, isDefaultRuntime && supportsDefaultRuntimeName)

		if !isDefaultRuntime {
			continue
		}

		if supportsDefaultRuntimeName {
			defaultRuntimePath := append(config.containerdPath(), "default_runtime")
			if config.GetPath(defaultRuntimePath) != nil {
				log.Warnf("The setting of default_runtime (%v) in containerd is deprecated", defaultRuntimePath)
			}
			continue
		}

		log.Warnf("Setting default_runtime is deprecated")
		defaultRuntimePath := append(config.containerdPath(), "default_runtime")
		config.initRuntime(defaultRuntimePath, o.runtimeType, "Runtime", runtimeBinary)
		config.initRuntime(defaultRuntimePath, o.runtimeType, "BinaryName", runtimeBinary)
	}
	return nil
}

// Revert performs a revert specific to v1 of the containerd config
func (config *configV1) Revert(o *options) error {
	defaultRuntimePath := append(config.containerdPath(), "default_runtime")
	defaultRuntimeOptionsPath := append(defaultRuntimePath, "options")
	if runtime, ok := config.GetPath(append(defaultRuntimeOptionsPath, "Runtime")).(string); ok {
		for _, runtimeBinary := range o.getRuntimeBinaries() {
			if path.Base(runtimeBinary) == path.Base(runtime) {
				config.DeletePath(append(defaultRuntimeOptionsPath, "Runtime"))
				break
			}
		}
	}
	if runtime, ok := config.GetPath(append(defaultRuntimeOptionsPath, "BinaryName")).(string); ok {
		for _, runtimeBinary := range o.getRuntimeBinaries() {
			if path.Base(runtimeBinary) == path.Base(runtime) {
				config.DeletePath(append(defaultRuntimeOptionsPath, "BinaryName"))
				break
			}
		}
	}

	if options, ok := config.GetPath(defaultRuntimeOptionsPath).(*toml.Tree); ok {
		if len(options.Keys()) == 0 {
			config.DeletePath(defaultRuntimeOptionsPath)
		}
	}

	if runtime, ok := config.GetPath(defaultRuntimePath).(*toml.Tree); ok {
		fields := []string{"runtime_type", "runtime_root", "runtime_engine", "privileged_without_host_devices"}
		if len(runtime.Keys()) <= len(fields) {
			matches := []string{}
			for _, f := range fields {
				e := runtime.Get(f)
				if e != nil {
					matches = append(matches, f)
				}
			}
			if len(matches) == len(runtime.Keys()) {
				for _, m := range matches {
					runtime.Delete(m)
				}
			}
		}
	}

	for i := 0; i < len(defaultRuntimePath); i++ {
		if runtimes, ok := config.GetPath(defaultRuntimePath[:len(defaultRuntimePath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				config.DeletePath(defaultRuntimePath[:len(defaultRuntimePath)-i])
			}
		}
	}

	for runtimeClass := range nvidiaRuntimeBinaries {
		config.revert(runtimeClass)
	}

	return nil
}
