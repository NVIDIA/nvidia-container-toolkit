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
	"io"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
)

const (
	configFileRelativePath = "nvidia-container-runtime/config.toml"
	configOverride         = "XDG_CONFIG_HOME"
	defaultConfigRoot      = "/etc"

	nvidiaContainerCliSection                       = "nvidia-container-cli"
	nvidiaContainerRuntimeConfigSection             = "nvidia-container-runtime"
	nvidiaContainerRuntimeExperimentalConfigSection = "nvidia-container-runtime.experimental"
)

type tomlConfig struct {
	logger   *log.Logger
	path     string
	sections []tomlSection
}

type tomlSection struct {
	section string
	keys    map[string]struct{}
}

func newDefaultConfigFileWithLogger(logger *log.Logger) configUpdater {
	configDir := defaultConfigRoot
	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		configDir = XDGConfigDir
	}
	configFilePath := filepath.Join(configDir, configFileRelativePath)

	return newConfigFromFileWithLogger(logger, configFilePath)
}

func newConfigFromFileWithLogger(logger *log.Logger, filepath string) configUpdater {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		logger.Warnf("The config file '%v' does not exist", filepath)
		return newNoopConfigUpdater()
	}

	sections := []tomlSection{
		{
			section: nvidiaContainerRuntimeConfigSection,
		},
		{
			section: nvidiaContainerRuntimeExperimentalConfigSection,
		},
		{
			section: nvidiaContainerCliSection,
			keys: map[string]struct{}{
				"root": {},
			},
		},
	}

	c := tomlConfig{
		logger:   logger,
		path:     filepath,
		sections: sections,
	}

	return &c
}

func (c tomlConfig) Update(cfg *Config) error {
	configFile, err := os.Open(c.path)
	if err != nil {
		return fmt.Errorf("error opening config file %v: %v", c.path, err)
	}
	defer configFile.Close()

	return c.updateFromReader(cfg, configFile)
}

func (c tomlConfig) updateFromReader(cfg *Config, reader io.Reader) error {
	toml, err := toml.LoadReader(reader)
	if err != nil {
		return fmt.Errorf("error reading TOML contents: %v", err)
	}

	for _, section := range c.sections {
		if v, ok := section.GetStringFrom(toml, "debug"); ok {
			cfg.DebugFilePath = v
		}

		if v, ok := section.GetStringFrom(toml, "runtime-path"); ok {
			cfg.RuntimePath = v
		}

		if v, ok := section.GetStringFrom(toml, "root"); ok {
			cfg.Root = v
		}

		if v, ok := section.GetStringFrom(toml, "log-level"); ok {
			cfg.LogLevel = v
		}
	}
	return nil
}

func (c tomlSection) GetStringFrom(toml *toml.Tree, key string) (string, bool) {
	value := c.GetFrom(toml, key)
	if value != nil {
		if v, ok := value.(string); ok {
			return v, ok
		}
	}
	return "", false
}

func (c tomlSection) GetFrom(toml *toml.Tree, key string) interface{} {
	if !c.validKey(key) {
		return nil
	}
	return toml.Get(c.configKey(key))
}

func (c tomlSection) validKey(key string) bool {
	if c.keys == nil {
		return true
	}

	_, exists := c.keys[key]

	return exists
}

func (c tomlSection) configKey(key string) string {
	if c.section == "" {
		return key
	}
	return c.section + "." + key
}
