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

package configure

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/ocihook"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

const (
	defaultRuntime = "docker"

	// defaultNVIDIARuntimeName is the default name to use in configs for the NVIDIA Container Runtime
	defaultNVIDIARuntimeName = "nvidia"
	// defaultNVIDIARuntimeExecutable is the default NVIDIA Container Runtime executable file name
	defaultNVIDIARuntimeExecutable          = "nvidia-container-runtime"
	defaultNVIDIARuntimeExpecutablePath     = "/usr/bin/nvidia-container-runtime"
	defaultNVIDIARuntimeHookExpecutablePath = "/usr/bin/nvidia-container-runtime-hook"

	defaultContainerdConfigFilePath = "/etc/containerd/config.toml"
	defaultCrioConfigFilePath       = "/etc/crio/crio.conf"
	defaultDockerConfigFilePath     = "/etc/docker/daemon.json"

	defaultContainerdDropInConfigFilePath = "/etc/containerd/conf.d/99-nvidia.toml"
	defaultCrioDropInConfigFilePath       = "/etc/crio/conf.d/99-nvidia.toml"

	defaultConfigSource = configSourceFile
	configSourceCommand = "command"
	configSourceFile    = "file"

	// TODO: We may want to spend some time unifying the handling of config
	// files here with the Setup-Cleanup logic in nvidia-ctk-installer.
	runtimeSpecificDefault = "RUNTIME_SPECIFIC_DEFAULT"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs a configure command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// config defines the options that can be set for the CLI through config files,
// environment variables, or command line config
type config struct {
	dryRun           bool
	runtime          string
	configFilePath   string
	dropInConfigPath string
	executablePath   string
	configSource     string
	mode             string
	hookFilePath     string

	nvidiaRuntime struct {
		name         string
		path         string
		hookPath     string
		setAsDefault bool
	}

	// cdi-specific options
	cdi struct {
		enabled bool
	}
}

func (m command) build() *cli.Command {
	// Create a config struct to hold the parsed environment variables or command line flags
	config := config{}

	// Create the 'configure' command
	configure := cli.Command{
		Name:  "configure",
		Usage: "Add a runtime to the specified container engine",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(&config)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.configureWrapper(&config)
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "update the runtime configuration as required but don't write changes to disk",
				Destination: &config.dryRun,
			},
			&cli.StringFlag{
				Name:        "runtime",
				Usage:       "the target runtime engine; one of [containerd, crio, docker]",
				Value:       defaultRuntime,
				Destination: &config.runtime,
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "path to the config file for the target runtime",
				Destination: &config.configFilePath,
			},
			&cli.StringFlag{
				Name:        "drop-in-config",
				Usage:       "path to the NVIDIA-specific config file to create. When specified, runtime configurations are saved to this file instead of modifying the main config file",
				Value:       runtimeSpecificDefault,
				Destination: &config.dropInConfigPath,
			},
			&cli.StringFlag{
				Name:        "executable-path",
				Usage:       "The path to the runtime executable. This is used to extract the current config",
				Destination: &config.executablePath,
			},
			&cli.StringFlag{
				Name:        "config-mode",
				Usage:       "the config mode for runtimes that support multiple configuration mechanisms",
				Destination: &config.mode,
			},
			&cli.StringFlag{
				Name:        "config-source",
				Usage:       "the source to retrieve the container runtime configuration; one of [command, file]\"",
				Destination: &config.configSource,
				Value:       defaultConfigSource,
			},
			&cli.StringFlag{
				Name:        "oci-hook-path",
				Usage:       "the path to the OCI runtime hook to create if --config-mode=oci-hook is specified. If no path is specified, the generated hook is output to STDOUT.\n\tNote: The use of OCI hooks is deprecated.",
				Destination: &config.hookFilePath,
			},
			&cli.StringFlag{
				Name:        "nvidia-runtime-name",
				Usage:       "specify the name of the NVIDIA runtime that will be added",
				Value:       defaultNVIDIARuntimeName,
				Destination: &config.nvidiaRuntime.name,
			},
			&cli.StringFlag{
				Name:        "nvidia-runtime-path",
				Aliases:     []string{"runtime-path"},
				Usage:       "specify the path to the NVIDIA runtime executable",
				Value:       defaultNVIDIARuntimeExecutable,
				Destination: &config.nvidiaRuntime.path,
			},
			&cli.StringFlag{
				Name:        "nvidia-runtime-hook-path",
				Usage:       "specify the path to the NVIDIA Container Runtime hook executable",
				Value:       defaultNVIDIARuntimeHookExpecutablePath,
				Destination: &config.nvidiaRuntime.hookPath,
			},
			&cli.BoolFlag{
				Name:        "nvidia-set-as-default",
				Aliases:     []string{"set-as-default"},
				Usage:       "set the NVIDIA runtime as the default runtime",
				Destination: &config.nvidiaRuntime.setAsDefault,
			},
			&cli.BoolFlag{
				Name:        "cdi.enabled",
				Aliases:     []string{"cdi.enable", "enable-cdi"},
				Usage:       "Enable CDI in the configured runtime",
				Destination: &config.cdi.enabled,
			},
		},
	}

	return &configure
}

func (m command) validateFlags(config *config) error {
	if config.mode == "oci-hook" || config.mode == "hook" {
		m.logger.Warningf("The %q config-mode is deprecated", config.mode)
		if !filepath.IsAbs(config.nvidiaRuntime.hookPath) {
			return fmt.Errorf("the NVIDIA runtime hook path %q is not an absolute path", config.nvidiaRuntime.hookPath)
		}
		return nil
	}
	if config.mode != "" && config.mode != "config-file" && config.mode != "config" {
		m.logger.Warningf("Ignoring unsupported config mode for %v: %q", config.runtime, config.mode)
	}
	config.mode = "config-file"

	switch config.runtime {
	case "containerd", "crio", "docker":
		break
	default:
		return fmt.Errorf("unrecognized runtime '%v'", config.runtime)
	}

	switch config.runtime {
	case "containerd", "crio":
		if config.nvidiaRuntime.path == defaultNVIDIARuntimeExecutable {
			config.nvidiaRuntime.path = defaultNVIDIARuntimeExpecutablePath
		}
		if !filepath.IsAbs(config.nvidiaRuntime.path) {
			return fmt.Errorf("the NVIDIA runtime path %q is not an absolute path", config.nvidiaRuntime.path)
		}
	}

	if config.runtime != "containerd" && config.runtime != "docker" {
		if config.cdi.enabled {
			m.logger.Warningf("Ignoring cdi.enabled flag for %v", config.runtime)
		}
		config.cdi.enabled = false
	}

	if config.executablePath != "" && config.runtime == "docker" {
		m.logger.Warningf("Ignoring executable-path=%q flag for %v", config.executablePath, config.runtime)
		config.executablePath = ""
	}

	switch config.configSource {
	case configSourceCommand:
		if config.runtime == "docker" {
			m.logger.Warningf("A %v Config Source is not supported for %v; using %v", config.configSource, config.runtime, configSourceFile)
			config.configSource = configSourceFile
		}
	case configSourceFile:
		break
	default:
		return fmt.Errorf("unrecognized Config Source: %v", config.configSource)
	}

	if config.configFilePath == "" {
		switch config.runtime {
		case "containerd":
			config.configFilePath = defaultContainerdConfigFilePath
		case "crio":
			config.configFilePath = defaultCrioConfigFilePath
		case "docker":
			config.configFilePath = defaultDockerConfigFilePath
		}
	}

	if config.dropInConfigPath == runtimeSpecificDefault {
		switch config.runtime {
		case "containerd":
			config.dropInConfigPath = defaultContainerdDropInConfigFilePath
		case "crio":
			config.dropInConfigPath = defaultCrioDropInConfigFilePath
		case "docker":
			config.dropInConfigPath = ""
		}
	}

	if config.dropInConfigPath != "" && config.runtime == "docker" {
		return fmt.Errorf("runtime %v does not support drop-in configs", config.runtime)
	}

	if config.dropInConfigPath != "" && !filepath.IsAbs(config.dropInConfigPath) {
		return fmt.Errorf("the drop-in-config path %q is not an absolute path", config.dropInConfigPath)
	}

	return nil
}

// configureWrapper updates the specified container engine config to enable the NVIDIA runtime
func (m command) configureWrapper(config *config) error {
	switch config.mode {
	case "oci-hook", "hook":
		return m.configureOCIHook(config)
	case "config-file", "config":
		return m.configureConfigFile(config)
	}
	return fmt.Errorf("unsupported config-mode: %v", config.mode)
}

// configureConfigFile updates the specified container engine config file to enable the NVIDIA runtime.
func (m command) configureConfigFile(config *config) error {
	configSource, err := config.resolveConfigSource()
	if err != nil {
		return err
	}

	var cfg engine.Interface
	switch config.runtime {
	case "containerd":
		cfg, err = containerd.New(
			containerd.WithLogger(m.logger),
			containerd.WithTopLevelConfigPath(config.configFilePath),
			containerd.WithConfigSource(configSource),
		)
	case "crio":
		cfg, err = crio.New(
			crio.WithLogger(m.logger),
			crio.WithTopLevelConfigPath(config.configFilePath),
			crio.WithConfigSource(configSource),
		)
	case "docker":
		cfg, err = docker.New(
			docker.WithLogger(m.logger),
			docker.WithPath(config.configFilePath),
		)
	default:
		err = fmt.Errorf("unrecognized runtime '%v'", config.runtime)
	}
	if err != nil || cfg == nil {
		return fmt.Errorf("unable to load config for runtime %v: %v", config.runtime, err)
	}

	err = cfg.AddRuntime(
		config.nvidiaRuntime.name,
		config.nvidiaRuntime.path,
		config.nvidiaRuntime.setAsDefault,
	)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	if config.cdi.enabled {
		cfg.EnableCDI()
	}

	outputPath := config.getOutputConfigPath()
	n, err := cfg.Save(outputPath)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	if outputPath != "" {
		if n == 0 {
			m.logger.Infof("Removed empty config from %v", outputPath)
		} else {
			m.logger.Infof("Wrote updated config to %v", outputPath)
		}
		m.logger.Infof("It is recommended that %v daemon be restarted.", config.runtime)
	}

	return nil
}

// resolveConfigSource returns the default config source or the user provided config source
func (c *config) resolveConfigSource() (toml.Loader, error) {
	switch c.configSource {
	case configSourceCommand:
		return c.getCommandConfigSource(), nil
	case configSourceFile:
		return toml.FromFile(c.configFilePath), nil
	default:
		return nil, fmt.Errorf("unrecognized config source: %s", c.configSource)
	}
}

// getConfigSourceCommand returns the default cli command to fetch the current runtime config
func (c *config) getCommandConfigSource() toml.Loader {
	switch c.runtime {
	case "containerd":
		return containerd.CommandLineSource("", c.executablePath)
	case "crio":
		return crio.CommandLineSource("", c.executablePath)
	}
	return toml.Empty
}

// getOutputConfigPath returns the configured config path or "" if dry-run is enabled
func (c *config) getOutputConfigPath() string {
	if c.dryRun {
		return engine.SaveToSTDOUT
	}
	if c.dropInConfigPath != "" {
		return c.dropInConfigPath
	}
	return c.configFilePath
}

// configureOCIHook creates and configures the OCI hook for the NVIDIA runtime
func (m *command) configureOCIHook(config *config) error {
	err := ocihook.CreateHook(config.hookFilePath, config.nvidiaRuntime.hookPath)
	if err != nil {
		return fmt.Errorf("error creating OCI hook: %v", err)
	}
	return nil
}
