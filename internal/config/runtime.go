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

// RuntimeConfig stores the config options for the NVIDIA Container Runtime
type RuntimeConfig struct {
	DebugFilePath string `toml:"debug"`
	// LogLevel defines the logging level for the application
	LogLevel string `toml:"log-level"`
	// Runtimes defines the candidates for the low-level runtime
	Runtimes []string    `toml:"runtimes"`
	Mode     string      `toml:"mode"`
	Modes    modesConfig `toml:"modes"`
}

// modesConfig defines (optional) per-mode configs
type modesConfig struct {
	CSV csvModeConfig `toml:"csv"`
	CDI cdiModeConfig `toml:"cdi"`
}

type cdiModeConfig struct {
	// SpecDirs allows for the default spec dirs for CDI to be overridden
	SpecDirs []string `toml:"spec-dirs"`
	// DefaultKind sets the default kind to be used when constructing fully-qualified CDI device names
	DefaultKind string `toml:"default-kind"`
	// AnnotationPrefixes sets the allowed prefixes for CDI annotation-based device injection
	AnnotationPrefixes []string `toml:"annotation-prefixes"`
}

type csvModeConfig struct {
	MountSpecPath string `toml:"mount-spec-path"`
}

// GetDefaultRuntimeConfig defines the default values for the config
func GetDefaultRuntimeConfig() (*RuntimeConfig, error) {
	cfg, err := GetDefault()
	if err != nil {
		return nil, err
	}

	return &cfg.NVIDIAContainerRuntimeConfig, nil
}
