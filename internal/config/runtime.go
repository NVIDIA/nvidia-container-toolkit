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
	"github.com/sirupsen/logrus"
)

const (
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"

	auto = "auto"
)

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
}

type csvModeConfig struct {
	MountSpecPath string `toml:"mount-spec-path"`
}

// dummy allows us to unmarshal only a RuntimeConfig from a *toml.Tree
type dummy struct {
	Runtime RuntimeConfig `toml:"nvidia-container-runtime"`
}

// getRuntimeConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getRuntimeConfigFrom(toml *toml.Tree) (*RuntimeConfig, error) {
	cfg := GetDefaultRuntimeConfig()

	if toml == nil {
		return cfg, nil
	}

	d := dummy{
		Runtime: *cfg,
	}

	if err := toml.Unmarshal(&d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal runtime config: %v", err)
	}

	return &d.Runtime, nil
}

// GetDefaultRuntimeConfig defines the default values for the config
func GetDefaultRuntimeConfig() *RuntimeConfig {
	c := RuntimeConfig{
		DebugFilePath: "/dev/null",
		LogLevel:      logrus.InfoLevel.String(),
		Runtimes: []string{
			dockerRuncExecutableName,
			runcExecutableName,
		},
		Mode: auto,
		Modes: modesConfig{
			CSV: csvModeConfig{
				MountSpecPath: "/etc/nvidia-container-runtime/host-files-for-container.d",
			},
		},
	}

	return &c
}
