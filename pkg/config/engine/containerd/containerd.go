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

package containerd

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// Config represents the containerd config
type Config struct {
	*toml.Tree
	Logger               logger.Interface
	RuntimeType          string
	ContainerAnnotations []string
	// UseLegacyConfig indicates whether a config file pre v1.3 should be generated.
	// For version 1 config prior to containerd v1.4 the default runtime was
	// specified in a containerd.runtimes.default_runtime section.
	// This was deprecated in v1.4 in favour of containerd.default_runtime_name.
	// Support for this section has been removed in v2.0.
	UseLegacyConfig bool
}

var _ engine.Interface = (*Config)(nil)

type containerdCfgRuntime struct {
	tree *toml.Tree
}

var _ engine.RuntimeConfig = (*containerdCfgRuntime)(nil)

// GetBinaryPath retrieves the path to the low-level runtime binary for a runtime.
// If no path is available, the empty string is returned.
func (c *containerdCfgRuntime) GetBinaryPath() string {
	if c == nil || c.tree == nil {
		return ""
	}

	binPath, _ := c.tree.GetPath([]string{"options", "BinaryName"}).(string)
	return binPath
}

// New creates a containerd config with the specified options
func New(opts ...Option) (engine.Interface, error) {
	b := &builder{
		runtimeType: defaultRuntimeType,
	}
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
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	cfg := &Config{
		Tree:                 tomlConfig,
		Logger:               b.logger,
		RuntimeType:          b.runtimeType,
		ContainerAnnotations: b.containerAnnotations,
		UseLegacyConfig:      b.useLegacyConfig,
	}

	version, err := cfg.parseVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config version: %v", err)
	}
	switch version {
	case 1:
		return (*ConfigV1)(cfg), nil
	case 2:
		return cfg, nil
	}

	return nil, fmt.Errorf("unsupported config version: %v", version)
}

// parseVersion returns the version of the config
func (c *Config) parseVersion() (int, error) {
	defaultVersion := 2
	// For legacy configs, we default to v1 configs.
	if c.UseLegacyConfig {
		defaultVersion = 1
	}

	switch v := c.Get("version").(type) {
	case nil:
		switch len(c.Keys()) {
		case 0: // No config exists, or the config file is empty, use version inferred from containerd
			return defaultVersion, nil
		default: // A config file exists, has content, and no version is set
			return 1, nil
		}
	case int64:
		return int(v), nil
	default:
		return -1, fmt.Errorf("unsupported type for version field: %v", v)
	}
}

func (c *Config) GetRuntimeConfig(name string) (engine.RuntimeConfig, error) {
	if c == nil || c.Tree == nil {
		return nil, fmt.Errorf("config is nil")
	}
	runtimeData := c.GetSubtreeByPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name})
	return &containerdCfgRuntime{
		tree: runtimeData,
	}, nil
}

// CommandLineSource returns the CLI-based containerd config loader
func CommandLineSource(hostRoot string) toml.Loader {
	return toml.FromCommandLine(chrootIfRequired(hostRoot, "containerd", "config", "dump")...)
}

func chrootIfRequired(hostRoot string, commandLine ...string) []string {
	if hostRoot == "" || hostRoot == "/" {
		return commandLine
	}

	return append([]string{"chroot", hostRoot}, commandLine...)
}
