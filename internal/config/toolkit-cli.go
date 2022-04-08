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

import "github.com/pelletier/go-toml"

// CTKConfig stores the config options for the NVIDIA Container Toolkit CLI (nvidia-ctk)
type CTKConfig struct {
	Path string `toml:"path"`
}

// getCTKConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getCTKConfigFrom(toml *toml.Tree) *CTKConfig {
	cfg := getDefaultCTKConfig()

	if toml == nil {
		return cfg
	}

	cfg.Path = toml.GetDefault("nvidia-ctk.path", cfg.Path).(string)

	return cfg
}

// getDefaultCTKConfig defines the default values for the config
func getDefaultCTKConfig() *CTKConfig {
	c := CTKConfig{
		Path: "nvidia-ctk",
	}

	return &c
}
