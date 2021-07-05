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
	log "github.com/sirupsen/logrus"
)

type defaultConfig struct{}

var _ configUpdater = (*defaultConfig)(nil)

func newDefaultConfig() configUpdater {
	c := defaultConfig{}
	return &c
}

// Update defines the the default values for the config options
func (c defaultConfig) Update(cfg *Config) error {
	*cfg = Config{
		DebugFilePath: "/dev/null",
		LogLevel:      log.InfoLevel.String(),
	}
	return nil
}

func getDefaultConfig() *Config {
	cfg := &Config{}

	defaultConfig{}.Update(cfg)

	return cfg
}
