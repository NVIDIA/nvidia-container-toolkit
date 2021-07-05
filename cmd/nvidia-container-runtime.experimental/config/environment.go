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
	"os"
	"strings"
)

const (
	debugFilePathEnvvarName = "NVIDIA_CONTAINER_RUNTIME_DEBUG"
	runtimePathEnvvarName   = "NVIDIA_CONTAINER_RUNTIME_PATH"
	rootEnvvarName          = "NVIDIA_CONTAINER_RUNTIME_ROOT"
	logLevelEnvvarName      = "NVIDIA_CONTAINER_RUNTIME_LOG_LEVEL"
)

type envConfig struct{}

func newConfigFromEnvironment() configUpdater {
	c := envConfig{}

	return &c
}

func (c envConfig) Update(cfg *Config) error {
	debugFilePathEnvvar, exists := os.LookupEnv(debugFilePathEnvvarName)
	if exists && strings.TrimSpace(debugFilePathEnvvar) != "" {
		cfg.DebugFilePath = debugFilePathEnvvar
	}

	runtimePathEnvvar, exists := os.LookupEnv(runtimePathEnvvarName)
	if exists && strings.TrimSpace(runtimePathEnvvar) != "" {
		cfg.RuntimePath = runtimePathEnvvar
	}

	rootEnvvar, exists := os.LookupEnv(rootEnvvarName)
	if exists && strings.TrimSpace(rootEnvvar) != "" {
		cfg.Root = rootEnvvar
	}

	logLevelEnvvar, exists := os.LookupEnv(logLevelEnvvarName)
	if exists && strings.TrimSpace(logLevelEnvvar) != "" {
		cfg.LogLevel = logLevelEnvvar
	}

	return nil
}
