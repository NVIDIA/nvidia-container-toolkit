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
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// Config represents the cri-o config
type Config struct {
	*toml.Tree
	Logger       logger.Interface
	nvidiaConfig string
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
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logger.New()
	}
	if b.configSource == nil {
		b.configSource = toml.FromFile(b.path)
	}

	tomlConfig, err := b.configSource.Load()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Tree:         tomlConfig,
		Logger:       b.logger,
		nvidiaConfig: b.nvidiaConfig,
	}
	return &cfg, nil
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

	// If using NVIDIA-specific configuration, handle file cleanup
	if c.nvidiaConfig != "" {
		// Check if all NVIDIA runtimes are being removed
		remainingNvidiaRuntimes := 0
		if runtimes := c.GetPath([]string{"crio", "runtime", "runtimes"}); runtimes != nil {
			if runtimesTree, ok := runtimes.(*toml.Tree); ok {
				for _, runtimeName := range runtimesTree.Keys() {
					if c.isNvidiaRuntime(runtimeName) && runtimeName != name {
						remainingNvidiaRuntimes++
					}
				}
			}
		}

		// If this is the last NVIDIA runtime, remove the NVIDIA config file
		if remainingNvidiaRuntimes == 0 {
			if err := os.Remove(c.nvidiaConfig); err != nil && !os.IsNotExist(err) {
				c.Logger.Warningf("Failed to remove NVIDIA config file %s: %v", c.nvidiaConfig, err)
			} else {
				c.Logger.Infof("Removed NVIDIA config file: %s", c.nvidiaConfig)
			}
			// Don't modify the in-memory tree when using NVIDIA-specific configuration
			return nil
		}
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

// Save writes the config to the specified path or NVIDIA-specific config file
func (c *Config) Save(path string) (int64, error) {
	if c.nvidiaConfig == "" {
		// Backward compatibility: save to main config
		return c.Tree.Save(path)
	}

	// Ensure directory for NVIDIA config file exists
	dir := filepath.Dir(c.nvidiaConfig)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory for NVIDIA config: %w", err)
	}

	// Save runtime configs to NVIDIA config file
	nvidiaConfig := c.extractRuntimeConfig()
	n, err := nvidiaConfig.Save(c.nvidiaConfig)
	if err != nil {
		return n, fmt.Errorf("failed to save NVIDIA config: %w", err)
	}

	// For CRI-O, we don't need to update the main config with imports
	// CRI-O automatically loads config files from the config directory
	c.Logger.Infof("Wrote NVIDIA runtime configuration to: %s", c.nvidiaConfig)
	return n, nil
}

// extractRuntimeConfig creates a new config tree with only runtime configurations
func (c *Config) extractRuntimeConfig() *toml.Tree {
	config, _ := toml.TreeFromMap(map[string]interface{}{})

	// Extract runtime configurations for NVIDIA runtimes
	if runtimes := c.GetPath([]string{"crio", "runtime", "runtimes"}); runtimes != nil {
		if runtimesTree, ok := runtimes.(*toml.Tree); ok {
			nvidiaRuntimes, _ := toml.TreeFromMap(map[string]interface{}{})
			for _, name := range runtimesTree.Keys() {
				if c.isNvidiaRuntime(name) {
					if runtime := runtimesTree.Get(name); runtime != nil {
						nvidiaRuntimes.Set(name, runtime)
					}
				}
			}
			if len(nvidiaRuntimes.Keys()) > 0 {
				config.SetPath([]string{"crio", "runtime", "runtimes"}, nvidiaRuntimes)
			}
		}
	}

	// Extract default runtime if it's one of ours
	if defaultRuntime, ok := c.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		if c.isNvidiaRuntime(defaultRuntime) {
			config.SetPath([]string{"crio", "runtime", "default_runtime"}, defaultRuntime)
		}
	}

	return config
}

// isNvidiaRuntime checks if the runtime name is an NVIDIA runtime
func (c *Config) isNvidiaRuntime(name string) bool {
	return name == "nvidia" || name == "nvidia-cdi" || name == "nvidia-legacy"
}

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
