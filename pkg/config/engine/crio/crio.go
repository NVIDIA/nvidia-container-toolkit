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

const (
	// defaultCrioDropInDir is the default directory for CRI-O drop-in configuration files
	defaultCrioDropInDir = "/etc/crio/crio.conf.d"
	// dropInFileName is the name of the NVIDIA runtime drop-in file
	dropInFileName = "99-nvidia.conf"
)

// Config represents the cri-o config
type Config struct {
	*toml.Tree
	NVConfig         *toml.Tree // For drop-in configuration
	Logger           logger.Interface
	baseConfigPath   string
	dropInConfigPath string
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

	var dropInConfigPath string
	if b.dropInDir != "" {
		dropInConfigPath = b.dropInDir
	} else {
		dropInConfigPath = filepath.Join(defaultCrioDropInDir, dropInFileName)
	}

	nvConfig, _ := toml.TreeFromMap(map[string]interface{}{})

	cfg := Config{
		Tree:             tomlConfig,
		NVConfig:         nvConfig,
		Logger:           b.logger,
		baseConfigPath:   b.path,
		dropInConfigPath: dropInConfigPath,
	}
	return &cfg, nil
}

func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	dropInConfig := c.NVConfig

	runtimeNamesForConfig := engine.GetLowLevelRuntimes(c)
	for _, r := range runtimeNamesForConfig {
		if options, ok := c.GetPath([]string{"crio", "runtime", "runtimes", r}).(*toml.Tree); ok {
			c.Logger.Debugf("using options from runtime %v: %v", r, options.String())
			// Parse and copy the options
			optionsCopy, _ := toml.Load(options.String())
			dropInConfig.SetPath([]string{"crio", "runtime", "runtimes", name}, optionsCopy)
			break
		}
	}

	dropInConfig.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_path"}, path)
	dropInConfig.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_type"}, "oci")

	// Set as default if requested
	if setAsDefault {
		dropInConfig.SetPath([]string{"crio", "runtime", "default_runtime"}, name)
	}

	c.NVConfig = dropInConfig
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

func (c *Config) RemoveRuntime(name string) error {
	if c == nil {
		return nil
	}

	if _, err := os.Stat(c.dropInConfigPath); os.IsNotExist(err) {
		c.Logger.Debugf("Drop-in file %s does not exist, nothing to remove", c.dropInConfigPath)
		return nil
	}

	if err := os.Remove(c.dropInConfigPath); err != nil {
		return fmt.Errorf("failed to remove drop-in file %s: %w", c.dropInConfigPath, err)
	}

	c.Logger.Infof("Removed drop-in configuration at %s", c.dropInConfigPath)
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

// Save saves the drop-in configuration to the specified path
func (c *Config) Save(path string) (int64, error) {
	// Create drop-in directory if it doesn't exist
	dropInDir := filepath.Dir(c.dropInConfigPath)
	if err := os.MkdirAll(dropInDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create drop-in directory %s: %w", dropInDir, err)
	}

	// Save the NVConfig to the drop-in file path
	return c.NVConfig.Save(c.dropInConfigPath)
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
