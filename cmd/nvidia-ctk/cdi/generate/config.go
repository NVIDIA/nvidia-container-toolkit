/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package generate

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"
)

// New sets the source for the config file path.
// It uses a string pointer to the config file path option.
func New(config *string) *configAsValueSource {
	return &configAsValueSource{
		source: altsrc.NewStringPtrSourcer(config),
	}
}

// configAsValueSource provides flag values from a TOML configuration file, acting as a factory for specific cli.ValueSource instances.
// It uses an altsrc.Sourcer to find the configuration file path and loads it on demand.
type configAsValueSource struct {
	sync.Mutex
	source altsrc.Sourcer
	*config.Toml
}

// ValueFrom returns a cli.ValueSource for the given key.
// This can be used in the Sources chain for a cli.Flag.
func (c *configAsValueSource) ValueFrom(key string) cli.ValueSource {
	return &configValueLookup{
		key:  key,
		from: c,
	}
}

func (c *configAsValueSource) Get(key string) interface{} {
	c.Lock()
	defer c.Unlock()

	if c.Toml == nil {
		if err := c.loadFromConfig(); err != nil {
			return nil
		}
	}

	if c.Toml == nil {
		return nil
	}

	return c.Toml.Get(key)
}

// loadFromConfig loads the config file if it has not already been loaded.
// If the config file path is not specified, it uses the default config file path.
func (c *configAsValueSource) loadFromConfig() error {
	if c.Toml != nil {
		return nil
	}

	configFilePath := c.source.SourceURI()
	if configFilePath == "" {
		configFilePath = config.GetConfigFilePath()
	}
	configToml, err := config.New(
		config.WithConfigFile(configFilePath),
	)
	if err != nil {
		return err
	}
	c.Toml = configToml

	return nil
}

// configValueLookup is a cli.ValueSource that looks up values from a configAsValueSource.
type configValueLookup struct {
	key  string
	from *configAsValueSource
}

// Lookup returns the value for the key from the config file.
func (c *configValueLookup) Lookup() (string, bool) {
	if c == nil || c.from == nil {
		return "", false
	}
	if value := c.from.Get(c.key); value != nil {
		return fmt.Sprintf("%v", value), true
	}

	return "", false
}

// GoString returns a Go-syntax representation of the configuration source.
func (c *configValueLookup) GoString() string {
	return c.String()
}

// String returns a string representation of the configuration source.
func (c *configValueLookup) String() string {
	if c.from == nil || c.from.source == nil {
		return "config file"
	}
	configFile := c.from.source.SourceURI()
	if configFile == "" {
		configFile = config.GetConfigFilePath()
	}
	if configFile == "" {
		return "config file"
	}
	return fmt.Sprintf("config file %q", configFile)
}
