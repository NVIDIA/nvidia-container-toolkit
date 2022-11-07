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
	"encoding/json"
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/runtime/nvidia"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/docker"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	defaultRuntime = "docker"

	defaultDockerConfigFilePath = "/etc/docker/daemon.json"
	defaultCrioConfigFilePath   = "/etc/crio/crio.conf"
)

type command struct {
	logger *logrus.Logger
}

// NewCommand constructs an configure command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
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
	nvidiaOptions  nvidia.Options
}

func (m command) build() *cli.Command {
	// Create a config struct to hold the parsed environment variables or command line flags
	config := config{}

	// Create the 'configure' command
	configure := cli.Command{
		Name:  "configure",
		Usage: "Add a runtime to the specified container engine",
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
			Usage:       "the target runtime engine. One of [crio, docker]",
			Value:       defaultRuntime,
			Destination: &config.runtime,
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "path to the config file for the target runtime",
			Destination: &config.configFilePath,
		},
		&cli.StringFlag{
			Name:        "nvidia-runtime-name",
			Usage:       "specify the name of the NVIDIA runtime that will be added",
			Value:       nvidia.RuntimeName,
			Destination: &config.nvidiaOptions.RuntimeName,
		},
		&cli.StringFlag{
			Name:        "runtime-path",
			Usage:       "specify the path to the NVIDIA runtime executable",
			Value:       nvidia.RuntimeExecutable,
			Destination: &config.nvidiaOptions.RuntimePath,
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Usage:       "set the specified runtime as the default runtime",
			Destination: &config.nvidiaOptions.SetAsDefault,
		},
	}

	return &configure
}

func (m command) configureWrapper(c *cli.Context, config *config) error {
	switch config.runtime {
	case "crio":
		return m.configureCrio(c, config)
	case "docker":
		return m.configureDocker(c, config)
	}

	return fmt.Errorf("unrecognized runtime '%v'", config.runtime)
}

// configureDocker updates the docker config to enable the NVIDIA Container Runtime
func (m command) configureDocker(c *cli.Context, config *config) error {
	configFilePath := config.configFilePath
	if configFilePath == "" {
		configFilePath = defaultDockerConfigFilePath
	}

	cfg, err := docker.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = docker.UpdateConfig(
		cfg,
		config.nvidiaOptions.RuntimeName,
		config.nvidiaOptions.RuntimePath,
		config.nvidiaOptions.SetAsDefault,
	)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	if config.dryRun {
		output, err := json.MarshalIndent(cfg, "", "    ")
		if err != nil {
			return fmt.Errorf("unable to convert to JSON: %v", err)
		}
		os.Stdout.WriteString(fmt.Sprintf("%s\n", output))
		return nil
	}
	err = docker.FlushConfig(cfg, configFilePath)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	m.logger.Infof("Wrote updated config to %v", configFilePath)
	m.logger.Infof("It is recommended that the docker daemon be restarted.")

	return nil
}

// configureCrio updates the crio config to enable the NVIDIA Container Runtime
func (m command) configureCrio(c *cli.Context, config *config) error {
	configFilePath := config.configFilePath
	if configFilePath == "" {
		configFilePath = defaultCrioConfigFilePath
	}

	cfg, err := crio.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = crio.UpdateConfig(
		cfg,
		config.nvidiaOptions.RuntimeName,
		config.nvidiaOptions.RuntimePath,
		config.nvidiaOptions.SetAsDefault,
	)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	if config.dryRun {
		output, err := toml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("unable to convert to TOML: %v", err)
		}
		os.Stdout.WriteString(fmt.Sprintf("%s\n", output))
		return nil
	}
	err = crio.FlushConfig(configFilePath, cfg)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	m.logger.Infof("Wrote updated config to %v", configFilePath)
	m.logger.Infof("It is recommended that the cri-o daemon be restarted.")

	return nil
}
