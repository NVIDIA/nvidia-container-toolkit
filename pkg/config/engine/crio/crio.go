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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// Config represents the cri-o config
type Config struct {
	*toml.Tree
	Logger logger.Interface
}

type crioRuntime struct {
	tree *toml.Tree
}

var _ engine.RuntimeConfig = (*crioRuntime)(nil)

// GetBinaryPath retrieves the path to the low-level runtime binary for a runtime.
// If no path is available, the empty string is returned.
func (c *crioRuntime) GetBinaryPath() string {
	if c.tree != nil {
		if binaryPath, ok := c.tree.GetPath([]string{"runtime_path"}).(string); ok {
			return binaryPath
		}
	}
	return ""
}

var _ engine.Interface = (*Config)(nil)

// New creates a cri-o config with the specified options
func New(opts ...Option) (engine.Interface, error) {
	b := &builder{
		logger: logger.New(),
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.configSource == nil {
		b.configSource = toml.FromFile(b.topLevelConfigPath)
	}

	sourceConfig, err := b.configSource.Load()
	if err != nil {
		return nil, err
	}

	var destinationConfig *toml.Tree
	if b.configDestination != nil {
		destinationConfig, err = b.configDestination.Load()
		if err != nil {
			return nil, err
		}
	} else {
		destinationConfig = toml.NewEmpty()
	}

	cfg := &engine.Config{
		Source: &Config{
			Tree:   sourceConfig,
			Logger: b.logger,
		},
		Destination: &Config{
			Tree:   destinationConfig,
			Logger: b.logger,
		},
	}

	return cfg, nil
}

// AddRuntime adds a new runtime to the crio config.
// The runtime options are extracted from the default runtime and the applicable
// settings are overridden.
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil || c.Tree == nil {
		return fmt.Errorf("config is nil")
	}
	defaultRuntimeOptions := c.GetDefaultRuntimeOptions()
	return c.AddRuntimeWithOptions(name, path, setAsDefault, defaultRuntimeOptions)
}

func (c *Config) GetDefaultRuntimeOptions() interface{} {
	runtimeNamesForConfig := engine.GetLowLevelRuntimes(c)
	for _, r := range runtimeNamesForConfig {
		options := c.GetSubtreeByPath([]string{"crio", "runtime", "runtimes", r})
		if options != nil {
			c.Logger.Debugf("Using options from runtime %v: %v", r, options)
			return options.Copy()
		}
	}
	c.Logger.Warningf("Could not infer options from runtimes %v", runtimeNamesForConfig)
	return nil
}

func (c *Config) AddRuntimeWithOptions(name string, path string, setAsDefault bool, options interface{}) error {
	config := *c.Tree

	if options != nil {
		config.SetPath([]string{"crio", "runtime", "runtimes", name}, options)
	}
	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_path"}, path)
	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_type"}, "oci")

	if setAsDefault {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, name)
	} else {
		if defaultRuntime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
			if defaultRuntime == name {
				config.DeletePath([]string{"crio", "runtime", "default_runtime"})
			}
		}
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

// UpdateDefaultRuntime updates the default runtime setting in the config.
// When action is 'set' the provided runtime name is set as the default.
// When action is 'unset' we make sure the provided runtime name is not
// the default.
func (c *Config) UpdateDefaultRuntime(name string, action string) error {
	if action != engine.UpdateActionSet && action != engine.UpdateActionUnset {
		return fmt.Errorf("invalid action %q, valid actions are %q and %q", action, engine.UpdateActionSet, engine.UpdateActionUnset)
	}

	if c == nil || c.Tree == nil {
		if action == engine.UpdateActionSet {
			return fmt.Errorf("config toml is nil")
		}
		return nil
	}

	config := *c.Tree

	if action == engine.UpdateActionSet {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, name)
	} else {
		if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
			if runtime == name {
				config.DeletePath([]string{"crio", "runtime", "default_runtime"})
			}
		}
	}

	*c.Tree = config
	return nil
}

func (c *Config) GetRuntimeConfig(name string) (engine.RuntimeConfig, error) {
	if c == nil || c.Tree == nil {
		return nil, fmt.Errorf("config is nil")
	}
	runtimeData := c.GetSubtreeByPath([]string{"crio", "runtime", "runtimes", name})
	return &crioRuntime{
		tree: runtimeData,
	}, nil
}

// EnableCDI is a no-op for CRI-O since it always enabled where supported.
func (c *Config) EnableCDI() {}

// CommandLineSource returns the CLI-based crio config loader
func CommandLineSource(hostRoot string, executablePath string) toml.Loader {
	if executablePath == "" {
		executablePath = "crio"
	}
	return toml.LoadFirst(
		toml.FromCommandLine(chrootIfRequired(hostRoot, executablePath, "status", "config")...),
		toml.FromCommandLine(chrootIfRequired(hostRoot, "crio-status", "config")...),
	)
}

func chrootIfRequired(hostRoot string, commandLine ...string) []string {
	if hostRoot == "" || hostRoot == "/" {
		return commandLine
	}

	return append([]string{"chroot", hostRoot}, commandLine...)
}
