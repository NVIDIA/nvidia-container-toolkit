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
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/ocihook"
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
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs an configure command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// config defines the options that can be set for the CLI through config files,
// environment variables, or command line config
type config struct {
	dryRun         bool
	runtime        string
	configFilePath string
	mode           string
	hookFilePath   string

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
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &config)
		},
		Action: func(c *cli.Context) error {
			return m.configureWrapper(c, &config)
		},
	}

	configure.Flags = []cli.Flag{
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
			Name:        "config-mode",
			Usage:       "the config mode for runtimes that support multiple configuration mechanisms",
			Destination: &config.mode,
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
			Usage:       "Enable CDI in the configured runtime",
			Destination: &config.cdi.enabled,
		},
	}

	return &configure
}

func (m command) validateFlags(c *cli.Context, config *config) error {
	if config.mode == "oci-hook" {
		if !filepath.IsAbs(config.nvidiaRuntime.hookPath) {
			return fmt.Errorf("the NVIDIA runtime hook path %q is not an absolute path", config.nvidiaRuntime.hookPath)
		}
		return nil
	}
	if config.mode != "" && config.mode != "config-file" {
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

	return nil
}

// configureWrapper updates the specified container engine config to enable the NVIDIA runtime
func (m command) configureWrapper(c *cli.Context, config *config) error {
	switch config.mode {
	case "oci-hook":
		return m.configureOCIHook(c, config)
	case "config-file":
		return m.configureConfigFile(c, config)
	}
	return fmt.Errorf("unsupported config-mode: %v", config.mode)
}

// configureConfigFile updates the specified container engine config file to enable the NVIDIA runtime.
func (m command) configureConfigFile(c *cli.Context, config *config) error {
	configFilePath := config.resolveConfigFilePath()

	var cfg engine.Interface
	var err error
	switch config.runtime {
	case "containerd":
		cfg, err = containerd.New(
			containerd.WithLogger(m.logger),
			containerd.WithPath(configFilePath),
		)
	case "crio":
		cfg, err = crio.New(
			crio.WithLogger(m.logger),
			crio.WithPath(configFilePath),
		)
	case "docker":
		cfg, err = docker.New(
			docker.WithLogger(m.logger),
			docker.WithPath(configFilePath),
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

	err = enableCDI(config, cfg)
	if err != nil {
		return fmt.Errorf("failed to enable CDI in %s: %w", config.runtime, err)
	}

	outputPath := config.getOuputConfigPath()
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

// resolveConfigFilePath returns the default config file path for the configured container engine
func (c *config) resolveConfigFilePath() string {
	if c.configFilePath != "" {
		return c.configFilePath
	}
	switch c.runtime {
	case "containerd":
		return defaultContainerdConfigFilePath
	case "crio":
		return defaultCrioConfigFilePath
	case "docker":
		return defaultDockerConfigFilePath
	}
	return ""
}

// getOuputConfigPath returns the configured config path or "" if dry-run is enabled
func (c *config) getOuputConfigPath() string {
	if c.dryRun {
		return ""
	}
	return c.resolveConfigFilePath()
}

// configureOCIHook creates and configures the OCI hook for the NVIDIA runtime
func (m *command) configureOCIHook(c *cli.Context, config *config) error {
	err := ocihook.CreateHook(config.hookFilePath, config.nvidiaRuntime.hookPath)
	if err != nil {
		return fmt.Errorf("error creating OCI hook: %v", err)
	}
	return nil
}

// enableCDI enables the use of CDI in the corresponding container engine
func enableCDI(config *config, cfg engine.Interface) error {
	if !config.cdi.enabled {
		return nil
	}
	switch config.runtime {
	case "containerd":
		cfg.Set("enable_cdi", true)
	case "docker":
		cfg.Set("experimental", true)
	default:
		return fmt.Errorf("enabling CDI in %s is not supported", config.runtime)
	}
	return nil
}
