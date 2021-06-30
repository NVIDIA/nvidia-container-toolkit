/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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

package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Config defines the configuration options for the NVIDIA Container Runtime
type Config struct {
	// Root defines the root of the file system to be used for locating mounts
	Root string
	// DebugFilePath defines a log file to print debug output to
	DebugFilePath string
	// RuntimePath defines the path to an OCI compliant runtime
	RuntimePath string
	// LogLevel defines the logging level for the application
	LogLevel string
}

//go:generate moq -stub -out config_mock.go . configUpdater

// configUpdate represents an interface for applying updates to a config.
type configUpdater interface {
	Update(*Config) error
}

// GetConfig returns a config struct with the values resolved. The values are defined in order of
// priority:
// 1. From the associated environment variables
// 2. From the loaded config file
// 3. From the default values defined in the `defaultConfig` function
func GetConfig(logger *log.Logger) (*Config, error) {
	cfg := &Config{}

	configs := newListWithLogger(logger,
		newDefaultConfig(),
		newDefaultConfigFileWithLogger(logger),
		newConfigFromEnvironment(),
	)

	err := configs.Update(cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting config: %v", err)
	}
	return cfg, nil
}
