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

package crio

import (
	"fmt"

	"github.com/pelletier/go-toml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
)

// Config represents the cri-o config
type Config struct {
	*toml.Tree
	Logger logger.Interface
}

var _ engine.Interface = (*Config)(nil)

// New creates a cri-o config with the specified options
func New(opts ...Option) (engine.Interface, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b.build()
}

// AddRuntime adds a new runtime to the crio config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	config := *c.Tree

	// By default we extract the runtime options from the runc settings; if this does not exist we get the options from the default runtime specified in the config.
	runtimeNamesForConfig := []string{"runc"}
	if name, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok && name != "" {
		runtimeNamesForConfig = append(runtimeNamesForConfig, name)
	}
	for _, r := range runtimeNamesForConfig {
		if options, ok := config.GetPath([]string{"crio", "runtime", "runtimes", r}).(*toml.Tree); ok {
			c.Logger.Debugf("using options from runtime %v: %v", r, options.String())
			options, _ = toml.Load(options.String())
			config.SetPath([]string{"crio", "runtime", "runtimes", name}, options)
			break
		}
	}

	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_path"}, path)
	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_type"}, "oci")

	if setAsDefault {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, name)
	}

	*c.Tree = config
	return nil
}

// DefaultRuntime returns the default runtime for the cri-o config
func (c *Config) DefaultRuntime() string {
	if c == nil || c.Tree == nil {
		return ""
	}
	if runtime, ok := c.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the cri-o config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil {
		return nil
	}

	config := *c.Tree
	if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		if runtime == name {
			config.DeletePath([]string{"crio", "runtime", "default_runtime"})
		}
	}

	runtimeClassPath := []string{"crio", "runtime", "runtimes", name}
	config.DeletePath(runtimeClassPath)
	for i := 0; i < len(runtimeClassPath); i++ {
		remainingPath := runtimeClassPath[:len(runtimeClassPath)-i]
		if entry, ok := config.GetPath(remainingPath).(*toml.Tree); ok {
			if len(entry.Keys()) != 0 {
				break
			}
			config.DeletePath(remainingPath)
		}
	}

	*c.Tree = config
	return nil
}

// Set sets the specified cri-o option.
func (c *Config) Set(key string, value interface{}) {
	config := *c.Tree
	config.Set(key, value)
	*c.Tree = config
}

// Save writes the config to the specified path
func (c Config) Save(path string) (int64, error) {
	config := c.Tree
	output, err := config.Marshal()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
	}

	n, err := engine.Config(path).Write(output)
	return int64(n), err
}
