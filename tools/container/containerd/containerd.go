/**
# Copyright (c) 2020-2021, NVIDIA CORPORATION.  All rights reserved.
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
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container"
)

const (
	defaultConfig        = "/etc/containerd/config.toml"
	defaultSocket        = "/run/containerd/containerd.sock"
	defaultRuntimeClass  = "nvidia"
	defaultRuntmeType    = "io.containerd.runc.v2"
	defaultSetAsDefault  = true
	defaultRestartMode   = "signal"
	defaultHostRootMount = "/host"
)

// options stores the configuration from the command line or environment variables
type options struct {
	container.Options

	// containerd-specific options
	useLegacyConfig bool
	runtimeType     string

	ContainerRuntimeModesCDIAnnotationPrefixes cli.StringSlice
}

func main() {
	options := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "containerd"
	c.Usage = "Update a containerd config with the nvidia-container-runtime"
	c.Version = info.GetVersionString()

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Trigger a containerd config to be updated"
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
	cleanup.Usage = "Trigger any updates made to a containerd config to be undone"
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
			Usage:       "Path to the containerd config file",
			Value:       defaultConfig,
			Destination: &options.Config,
			EnvVars:     []string{"RUNTIME_CONFIG", "CONTAINERD_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the containerd socket file",
			Value:       defaultSocket,
			Destination: &options.Socket,
			EnvVars:     []string{"RUNTIME_SOCKET", "CONTAINERD_SOCKET"},
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how containerd should be restarted;  If 'none' is selected, it will not be restarted [signal | systemd | none]",
			Value:       defaultRestartMode,
			Destination: &options.RestartMode,
			EnvVars:     []string{"RUNTIME_RESTART_MODE", "CONTAINERD_RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"nvidia-runtime-name", "runtime-class"},
			Usage:       "The name of the runtime class to set for the nvidia-container-runtime",
			Value:       defaultRuntimeClass,
			Destination: &options.RuntimeName,
			EnvVars:     []string{"NVIDIA_RUNTIME_NAME", "CONTAINERD_RUNTIME_CLASS"},
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
			Usage:       "Set nvidia-container-runtime as the default runtime",
			Value:       defaultSetAsDefault,
			Destination: &options.SetAsDefault,
			EnvVars:     []string{"NVIDIA_RUNTIME_SET_AS_DEFAULT", "CONTAINERD_SET_AS_DEFAULT"},
			Hidden:      true,
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting containerd using systemd",
			Value:       defaultHostRootMount,
			Destination: &options.HostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
		},
		&cli.BoolFlag{
			Name:        "use-legacy-config",
			Usage:       "Specify whether a legacy (pre v1.3) config should be used",
			Destination: &options.useLegacyConfig,
			EnvVars:     []string{"CONTAINERD_USE_LEGACY_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "runtime-type",
			Usage:       "The runtime_type to use for the configured runtime classes",
			Value:       defaultRuntmeType,
			Destination: &options.runtimeType,
			EnvVars:     []string{"CONTAINERD_RUNTIME_TYPE"},
		},
		&cli.StringSliceFlag{
			Name:        "nvidia-container-runtime-modes.cdi.annotation-prefixes",
			Destination: &options.ContainerRuntimeModesCDIAnnotationPrefixes,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODES_CDI_ANNOTATION_PREFIXES"},
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Fatal(fmt.Errorf("Error: %v", err))
	}
}

// Setup updates a containerd configuration to include the nvidia-containerd-runtime and reloads it
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	cfg, err := containerd.New(
		containerd.WithPath(o.Config),
		containerd.WithRuntimeType(o.runtimeType),
		containerd.WithUseLegacyConfig(o.useLegacyConfig),
		containerd.WithContainerAnnotations(o.containerAnnotationsFromCDIPrefixes()...),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Configure(cfg)
	if err != nil {
		return fmt.Errorf("unable to configure containerd: %v", err)
	}

	err = RestartContainerd(o)
	if err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}

	log.Infof("Completed 'setup' for %v", c.App.Name)

	return nil
}

// Cleanup reverts a containerd configuration to remove the nvidia-containerd-runtime and reloads it
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	cfg, err := containerd.New(
		containerd.WithPath(o.Config),
		containerd.WithRuntimeType(o.runtimeType),
		containerd.WithUseLegacyConfig(o.useLegacyConfig),
		containerd.WithContainerAnnotations(o.containerAnnotationsFromCDIPrefixes()...),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Unconfigure(cfg)
	if err != nil {
		return fmt.Errorf("unable to unconfigure containerd: %v", err)
	}

	err = RestartContainerd(o)
	if err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}

	log.Infof("Completed 'cleanup' for %v", c.App.Name)

	return nil
}

// RestartContainerd restarts containerd depending on the value of restartModeFlag
func RestartContainerd(o *options) error {
	return o.Restart("containerd", SignalContainerd)
}

// containerAnnotationsFromCDIPrefixes returns the container annotations to set for the given CDI prefixes.
func (o *options) containerAnnotationsFromCDIPrefixes() []string {
	var annotations []string
	for _, prefix := range o.ContainerRuntimeModesCDIAnnotationPrefixes.Value() {
		annotations = append(annotations, prefix+"*")
	}

	return annotations
}
