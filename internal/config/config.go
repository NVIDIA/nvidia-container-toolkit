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
	"io"
	"os"
	"path"

	"github.com/pelletier/go-toml"
)

const (
	configOverride = "XDG_CONFIG_HOME"
	configFilePath = "nvidia-container-runtime/config.toml"
)

var (
	// DefaultExecutableDir specifies the default path to use for executables if they cannot be located in the path.
	DefaultExecutableDir = "/usr/bin"

	// NVIDIAContainerRuntimeHookExecutable is the executable name for the NVIDIA Container Runtime Hook
	NVIDIAContainerRuntimeHookExecutable = "nvidia-container-runtime-hook"
	// NVIDIAContainerToolkitExecutable is the executable name for the NVIDIA Container Toolkit (an alias for the NVIDIA Container Runtime Hook)
	NVIDIAContainerToolkitExecutable = "nvidia-container-toolkit"

	configDir = "/etc/"
)

// Config represents the contents of the config.toml file for the NVIDIA Container Toolkit
// Note: This is currently duplicated by the HookConfig in cmd/nvidia-container-toolkit/hook_config.go
type Config struct {
	NVIDIAContainerCLIConfig     ContainerCLIConfig `toml:"nvidia-container-cli"`
	NVIDIACTKConfig              CTKConfig          `toml:"nvidia-ctk"`
	NVIDIAContainerRuntimeConfig RuntimeConfig      `toml:"nvidia-container-runtime"`
}

// GetConfig sets up the config struct. Values are read from a toml file
// or set via the environment.
func GetConfig() (*Config, error) {
	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		configDir = XDGConfigDir
	}

	configFilePath := path.Join(configDir, configFilePath)

	tomlFile, err := os.Open(configFilePath)
	if err != nil {
		return getDefaultConfig(), nil
	}
	defer tomlFile.Close()

	cfg, err := loadConfigFrom(tomlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config values: %v", err)
	}

	return cfg, nil
}

// loadRuntimeConfigFrom reads the config from the specified Reader
func loadConfigFrom(reader io.Reader) (*Config, error) {
	toml, err := toml.LoadReader(reader)
	if err != nil {
		return nil, err
	}

	return getConfigFrom(toml)
}

// getConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getConfigFrom(toml *toml.Tree) (*Config, error) {
	cfg := getDefaultConfig()

	if toml == nil {
		return cfg, nil
	}

	cfg.NVIDIAContainerCLIConfig = *getContainerCLIConfigFrom(toml)
	cfg.NVIDIACTKConfig = *getCTKConfigFrom(toml)
	runtimeConfig, err := getRuntimeConfigFrom(toml)
	if err != nil {
		return nil, fmt.Errorf("failed to load nvidia-container-runtime config: %v", err)
	}
	cfg.NVIDIAContainerRuntimeConfig = *runtimeConfig

	return cfg, nil
}

// getDefaultConfig defines the default values for the config
func getDefaultConfig() *Config {
	c := Config{
		NVIDIAContainerCLIConfig:     *getDefaultContainerCLIConfig(),
		NVIDIACTKConfig:              *getDefaultCTKConfig(),
		NVIDIAContainerRuntimeConfig: *GetDefaultRuntimeConfig(),
	}

	return &c
}
