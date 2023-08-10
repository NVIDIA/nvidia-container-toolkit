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

// RuntimeHookConfig stores the config options for the NVIDIA Container Runtime
type RuntimeHookConfig struct {
	// Path specifies the path to the NVIDIA Container Runtime hook binary.
	// If an executable name is specified, this will be resolved in the path.
	Path string `toml:"path"`
	// SkipModeDetection disables the mode check for the runtime hook.
	SkipModeDetection bool `toml:"skip-mode-detection"`
}

// GetDefaultRuntimeHookConfig defines the default values for the config
func GetDefaultRuntimeHookConfig() (*RuntimeHookConfig, error) {
	cfg, err := GetDefault()
	if err != nil {
		return nil, err
	}

	return &cfg.NVIDIAContainerRuntimeHookConfig, nil
}
