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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"
)

// Config returns a configAsValueSource for the generate options.
// This allows for flags to be specified in a config file.
func (o *options) Config() *configAsValueSource {
	if o.appliedConfig == nil {
		o.appliedConfig = &configAsValueSource{}
	}
	return o.appliedConfig.From(&o.config)
}

// configAsValueSource provides flag values from a TOML configuration file, acting as a factory for specific cli.ValueSource instances.
// It uses an altsrc.Sourcer to find the configuration file path and loads it on demand.
type configAsValueSource struct {
	sync.Mutex
	source altsrc.Sourcer
	*config.Toml
}

// configValueLookup is a cli.ValueSource that looks up values from a configAsValueSource.
type configValueLookup struct {
	key  string
	from *configAsValueSource
}

// From sets the source for the config file path.
// It uses a string pointer to the config file path option.
func (c *configAsValueSource) From(config *string) *configAsValueSource {
	c.Lock()
	defer c.Unlock()
	if c.source == nil {
		c.source = altsrc.NewStringPtrSourcer(config)
	}
	return c
}

// loadIfRequired loads the config file if it has not already been loaded.
// If the config file path is not specified, it uses the default config file path.
func (c *configAsValueSource) loadIfRequired() error {
	c.Lock()
	defer c.Unlock()

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

// GetAsString gets the specified key from the config file as a string.
// It returns the value and a boolean indicating if the value was found.
func (c *configAsValueSource) GetAsString(key string) (string, bool) {
	if c == nil {
		return "", false
	}
	if err := c.loadIfRequired(); err != nil {
		return "", false
	}

	value := c.Get(key)
	if key == value {
		return "", false
	}

	return fmt.Sprintf("%v", value), true
}

// ValueSource returns a cli.ValueSource for the given key.
// This can be used in the Sources chain for a cli.Flag.
func (c *configAsValueSource) ValueSource(key string) cli.ValueSource {
	return &configValueLookup{
		key:  key,
		from: c,
	}
}

// Lookup returns the value for the key from the config file.
func (c *configValueLookup) Lookup() (string, bool) {
	return c.from.GetAsString(c.key)
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
