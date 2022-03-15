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
	configDir = "/etc/"
)

// RuntimeConfig stores the config options for the NVIDIA Container Runtime
type RuntimeConfig struct {
	DebugFilePath string
	Experimental  bool
}

// GetRuntimeConfig sets up the config struct. Values are read from a toml file
// or set via the environment.
func GetRuntimeConfig() (*RuntimeConfig, error) {
	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		configDir = XDGConfigDir
	}

	configFilePath := path.Join(configDir, configFilePath)

	tomlFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %v: %v", configFilePath, err)
	}
	defer tomlFile.Close()

	cfg, err := getRuntimeConfigFrom(tomlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config values: %v", err)
	}

	return cfg, nil
}

// getRuntimeConfigFrom reads the config from the specified Reader
func getRuntimeConfigFrom(reader io.Reader) (*RuntimeConfig, error) {
	toml, err := toml.LoadReader(reader)
	if err != nil {
		return nil, err
	}

	cfg := getDefaultRuntimeConfig()

	cfg.DebugFilePath = toml.GetDefault("nvidia-container-runtime.debug", cfg.DebugFilePath).(string)
	cfg.Experimental = toml.GetDefault("nvidia-container-runtime.experimental", cfg.Experimental).(bool)

	return cfg, nil
}

// getDefaultRuntimeConfig defines the default values for the config
func getDefaultRuntimeConfig() *RuntimeConfig {
	c := RuntimeConfig{
		DebugFilePath: "/dev/null",
		Experimental:  false,
	}

	return &c
}
