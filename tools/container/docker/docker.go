/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container"
)

const (
	defaultConfig       = "/etc/docker/daemon.json"
	defaultSocket       = "/var/run/docker.sock"
	defaultSetAsDefault = true
	// defaultRuntimeName specifies the NVIDIA runtime to be use as the default runtime if setting the default runtime is enabled
	defaultRuntimeName   = "nvidia"
	defaultRestartMode   = "signal"
	defaultHostRootMount = "/host"
)

// options stores the configuration from the command line or environment variables
type options struct {
	container.Options
}

func main() {
	options := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "docker"
	c.Usage = "Update docker config with the nvidia runtime"
	c.Version = info.GetVersionString()

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Trigger docker config to be updated"
	setup.ArgsUsage = "<runtime_dirname>"
	setup.Action = func(c *cli.Context) error {
		return Setup(c, &options)
	}
	setup.Before = func(c *cli.Context) error {
		return container.ParseArgs(c, &options.Options)
	}

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Trigger any updates made to docker config to be undone"
	cleanup.ArgsUsage = "<runtime_dirname>"
	cleanup.Action = func(c *cli.Context) error {
		return Cleanup(c, &options)
	}
	cleanup.Before = func(c *cli.Context) error {
		return container.ParseArgs(c, &options.Options)
	}

	// Register the subcommands with the top-level CLI
	c.Commands = []*cli.Command{
		&setup,
		&cleanup,
	}

	// Setup common flags across both subcommands. All subcommands get the same
	// set of flags even if they don't use some of them. This is so that we
	// only require the user to specify one set of flags for both 'startup'
	// and 'cleanup' to simplify things.
	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Path to docker config file",
			Value:       defaultConfig,
			Destination: &options.Config,
			EnvVars:     []string{"RUNTIME_CONFIG", "DOCKER_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the docker socket file",
			Value:       defaultSocket,
			Destination: &options.Socket,
			EnvVars:     []string{"RUNTIME_SOCKET", "DOCKER_SOCKET"},
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how docker should be restarted; If 'none' is selected it will not be restarted [signal | systemd | none ]",
			Value:       defaultRestartMode,
			Destination: &options.RestartMode,
			EnvVars:     []string{"RUNTIME_RESTART_MODE", "DOCKER_RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting docker using systemd",
			Value:       defaultHostRootMount,
			Destination: &options.HostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
			// Restart using systemd is currently not supported.
			// We hide this option for the time being.
			Hidden: true,
		},
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"nvidia-runtime-name", "runtime-class"},
			Usage:       "Specify the name of the `nvidia` runtime. If set-as-default is selected, the runtime is used as the default runtime.",
			Value:       defaultRuntimeName,
			Destination: &options.RuntimeName,
			EnvVars:     []string{"NVIDIA_RUNTIME_NAME", "DOCKER_RUNTIME_NAME"},
		},
		&cli.StringFlag{
			Name:        "nvidia-runtime-dir",
			Aliases:     []string{"runtime-dir"},
			Usage:       "The path where the nvidia-container-runtime binaries are located. If this is not specified, the first argument will be used instead",
			Destination: &options.RuntimeDir,
			EnvVars:     []string{"NVIDIA_RUNTIME_DIR"},
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Usage:       "Set the `nvidia` runtime as the default runtime.",
			Value:       defaultSetAsDefault,
			Destination: &options.SetAsDefault,
			EnvVars:     []string{"NVIDIA_RUNTIME_SET_AS_DEFAULT", "DOCKER_SET_AS_DEFAULT"},
			Hidden:      true,
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Errorf("Error running docker configuration: %v", err)
		os.Exit(1)
	}
}

// Setup updates docker configuration to include the nvidia runtime and reloads it
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	cfg, err := docker.New(
		docker.WithPath(o.Config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Configure(cfg)
	if err != nil {
		return fmt.Errorf("unable to configure docker: %v", err)
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to restart docker: %v", err)
	}

	log.Infof("Completed 'setup' for %v", c.App.Name)

	return nil
}

// Cleanup reverts docker configuration to remove the nvidia runtime and reloads it
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	cfg, err := docker.New(
		docker.WithPath(o.Config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Unconfigure(cfg)
	if err != nil {
		return fmt.Errorf("unable to unconfigure docker: %v", err)
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to signal docker: %v", err)
	}

	log.Infof("Completed 'cleanup' for %v", c.App.Name)

	return nil
}

// RestartDocker restarts docker depending on the value of restartModeFlag
func RestartDocker(o *options) error {
	return o.Restart("docker", SignalDocker)
}
