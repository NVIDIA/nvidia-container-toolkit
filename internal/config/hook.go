/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package config

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// RuntimeHookConfig stores the config options for the NVIDIA Container Runtime
type RuntimeHookConfig struct {
	// Path specifies the path to the NVIDIA Container Runtime hook binary.
	// If an executable name is specified, this will be resolved in the path.
	Path string `toml:"path"`
	// SkipModeDetection disables the mode check for the runtime hook.
	SkipModeDetection bool `toml:"skip-mode-detection"`
}

// dummyHookConfig allows us to unmarshal only a RuntimeHookConfig from a *toml.Tree
type dummyHookConfig struct {
	RuntimeHook RuntimeHookConfig `toml:"nvidia-container-runtime-hook"`
}

// getRuntimeHookConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getRuntimeHookConfigFrom(toml *toml.Tree) (*RuntimeHookConfig, error) {
	cfg := GetDefaultRuntimeHookConfig()

	if toml == nil {
		return cfg, nil
	}

	d := dummyHookConfig{
		RuntimeHook: *cfg,
	}

	if err := toml.Unmarshal(&d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal runtime config: %v", err)
	}

	return &d.RuntimeHook, nil
}

// GetDefaultRuntimeHookConfig defines the default values for the config
func GetDefaultRuntimeHookConfig() *RuntimeHookConfig {
	c := RuntimeHookConfig{
		Path:              NVIDIAContainerRuntimeHookExecutable,
		SkipModeDetection: false,
	}

	return &c
}
