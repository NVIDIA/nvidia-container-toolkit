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
**/

package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/hook"
	infoCLI "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/info"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/runtime"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/system"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

	cli "github.com/urfave/cli/v3"
)

// options defines the options that can be set for the CLI through config files,
// environment variables, or command line flags
type options struct {
	// Debug indicates whether the CLI is started in "debug" mode
	Debug bool
	// Quiet indicates whether the CLI is started in "quiet" mode
	Quiet bool
	// Config specifies the path to the config file
	Config string
}

func main() {
	log := logrus.New()

	// Create a options struct to hold the parsed environment variables or command line flags
	opts := options{}

	// Create the top-level CLI
	c := cli.Command{
		DisableSliceFlagSeparator: true,
		Name:                      "NVIDIA Container Toolkit CLI",
		UseShortOptionHandling:    true,
		EnableShellCompletion:     true,
		Usage:                     "Tools to configure the NVIDIA Container Toolkit",
		Version:                   info.GetVersionString(),
		// Set log-level for all subcommands
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			logLevel := logrus.InfoLevel
			if opts.Debug {
				logLevel = logrus.DebugLevel
			}
			if opts.Quiet {
				logLevel = logrus.ErrorLevel
			}
			log.SetLevel(logLevel)

			return ctx, nil
		},
		// Define the subcommands
		Commands: getCommands(logger.FromLogrus(log), &opts.Config),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "Enable debug-level logging",
				Destination: &opts.Debug,
				Sources:     cli.EnvVars("NVIDIA_CTK_DEBUG"),
			},
			&cli.BoolFlag{
				Name:        "quiet",
				Usage:       "Suppress all output except for errors; overrides --debug",
				Destination: &opts.Quiet,
				Sources:     cli.EnvVars("NVIDIA_CTK_QUIET"),
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Path to the config file",
				Destination: &opts.Config,
				Sources:     cli.EnvVars("NVIDIA_CTK_CONFIG"),
			},
		},
	}

	// Run the CLI
	err := c.Run(context.Background(), os.Args)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
}

func getCommands(logger logger.Interface, configFilePath *string) []*cli.Command {
	return []*cli.Command{
		hook.NewCommand(logger),
		runtime.NewCommand(logger),
		infoCLI.NewCommand(logger),
		cdi.NewCommand(logger, configFilePath),
		system.NewCommand(logger),
		config.NewCommand(logger),
	}
}
