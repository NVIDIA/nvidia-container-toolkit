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

const (
	defaultConfigVersion = 2
	defaultRuntimeType   = "io.containerd.runc.v2"
)

// Config represents the containerd config
type Config struct {
	*toml.Tree
	configOptions
}

type configOptions struct {
	Version              int64
	Logger               logger.Interface
	RuntimeType          string
	ContainerAnnotations []string
	// UseLegacyConfig indicates whether a config file pre v1.3 should be generated.
	// For version 1 config prior to containerd v1.4 the default runtime was
	// specified in a containerd.runtimes.default_runtime section.
	// This was deprecated in v1.4 in favour of containerd.default_runtime_name.
	// Support for this section has been removed in v2.0.
	UseLegacyConfig bool
	// CRIRuntimePluginName represents the fully qualified name of the containerd plugin
	// for the CRI runtime service. The name of this plugin was changed in v3 of the
	// containerd configuration file.
	CRIRuntimePluginName string
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
		configVersion: defaultConfigVersion,
		runtimeType:   defaultRuntimeType,
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logger.New()
	}
	if b.configSource == nil {
		b.configSource = toml.FromFile(b.topLevelConfigPath)
	}

	sourceConfigTree, err := b.configSource.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	configVersion, err := b.parseVersion(sourceConfigTree)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config version: %w", err)
	}
	b.logger.Infof("Using config version %v", configVersion)

	criRuntimePluginName, err := b.criRuntimePluginName(configVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get CRI runtime plugin name: %w", err)
	}
	b.logger.Infof("Using CRI runtime plugin name %q", criRuntimePluginName)

	sourceConfigOptions := configOptions{
		Version:              configVersion,
		CRIRuntimePluginName: criRuntimePluginName,
		Logger:               b.logger,
		RuntimeType:          b.runtimeType,
		UseLegacyConfig:      b.useLegacyConfig,
		ContainerAnnotations: b.containerAnnotations,
	}
	sourceConfig := &Config{
		Tree:          sourceConfigTree,
		configOptions: sourceConfigOptions,
	}
	switch configVersion {
	case 1:
		// Version 1 configs do not support imports / drop-in files.
		// We return the sourceConfig as is.
		return (*ConfigV1)(sourceConfig), nil
	default:
		// For other versions, we create a DropInConfig with a reference to the
		// top-level config if present.
		topLevelConfig := &Config{
			Tree: func() *toml.Tree {
				t, _ := toml.FromFile(b.topLevelConfigPath).Load()
				return t
			}(),
			configOptions: sourceConfigOptions,
		}
		dropInConfig := &engine.Config{
			Source: sourceConfig,
			// The destinationConfig is an empty config with the same options
			// as the source config.
			Destination: &Config{
				Tree:          toml.NewEmpty(),
				configOptions: sourceConfigOptions,
			},
		}

		cfg := NewConfigWithDropIn(b.topLevelConfigPath, topLevelConfig, dropInConfig)
		return cfg, nil
	}
}

// parseVersion returns the version of the config
func (b *builder) parseVersion(c *toml.Tree) (int64, error) {
	if c == nil || len(c.Keys()) == 0 {
		// No config exists, or the config file is empty.
		if b.useLegacyConfig {
			// If a legacy config is explicitly requested, we default to a v1 config.
			return 1, nil
		}
		// Use the requested version.
		return int64(b.configVersion), nil
	}

	switch v := c.Get("version").(type) {
	case nil:
		return 1, nil
	case int64:
		return v, nil
	default:
		return -1, fmt.Errorf("unsupported type for version field: %v", v)
	}
}

func (b *builder) criRuntimePluginName(configVersion int64) (string, error) {
	switch configVersion {
	case 1:
		return "cri", nil
	case 2:
		return "io.containerd.grpc.v1.cri", nil
	default:
		return "io.containerd.cri.v1.runtime", nil
	}
}

func (c *Config) GetRuntimeConfig(name string) (engine.RuntimeConfig, error) {
	if c == nil || c.Tree == nil {
		return nil, fmt.Errorf("config is nil")
	}
	runtimeData := c.GetSubtreeByPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name})
	return &containerdCfgRuntime{
		tree: runtimeData,
	}, nil
}

// CommandLineSource returns the CLI-based containerd config loader
func CommandLineSource(hostRoot string, executablePath string) toml.Loader {
	if executablePath == "" {
		executablePath = "containerd"
	}
	return toml.FromCommandLine(chrootIfRequired(hostRoot, executablePath, "config", "dump")...)
}

func chrootIfRequired(hostRoot string, commandLine ...string) []string {
	if hostRoot == "" || hostRoot == "/" {
		return commandLine
	}

	return append([]string{"chroot", hostRoot}, commandLine...)
}
