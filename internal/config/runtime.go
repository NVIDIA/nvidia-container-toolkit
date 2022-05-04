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
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
)

const (
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
)

// RuntimeConfig stores the config options for the NVIDIA Container Runtime
type RuntimeConfig struct {
	DebugFilePath string
	Experimental  bool
	DiscoverMode  string
	// LogLevel defines the logging level for the application
	LogLevel string
	// Runtimes defines the candidates for the low-level runtime
	Runtimes []string
}

// getRuntimeConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getRuntimeConfigFrom(toml *toml.Tree) *RuntimeConfig {
	cfg := GetDefaultRuntimeConfig()

	if toml == nil {
		return cfg
	}

	cfg.DebugFilePath = toml.GetDefault("nvidia-container-runtime.debug", cfg.DebugFilePath).(string)
	cfg.Experimental = toml.GetDefault("nvidia-container-runtime.experimental", cfg.Experimental).(bool)
	cfg.DiscoverMode = toml.GetDefault("nvidia-container-runtime.discover-mode", cfg.DiscoverMode).(string)
	cfg.LogLevel = toml.GetDefault("nvidia-container-runtime.log-level", cfg.LogLevel).(string)

	configRuntimes := toml.Get("nvidia-container-runtime.runtimes")
	if configRuntimes != nil {
		var runtimes []string
		for _, r := range configRuntimes.([]interface{}) {
			runtimes = append(runtimes, r.(string))
		}
		cfg.Runtimes = runtimes
	}

	return cfg
}

// GetDefaultRuntimeConfig defines the default values for the config
func GetDefaultRuntimeConfig() *RuntimeConfig {
	c := RuntimeConfig{
		DebugFilePath: "/dev/null",
		Experimental:  false,
		DiscoverMode:  "auto",
		LogLevel:      logrus.InfoLevel.String(),
		Runtimes: []string{
			dockerRuncExecutableName,
			runcExecutableName,
		},
	}

	return &c
}
