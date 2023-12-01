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
	"os"

	"github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/hook"
	infoCLI "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/info"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/runtime"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/system"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"

	cli "github.com/urfave/cli/v2"
)

// options defines the options that can be set for the CLI through config files,
// environment variables, or command line flags
type options struct {
	// Debug indicates whether the CLI is started in "debug" mode
	Debug bool
	// Quiet indicates whether the CLI is started in "quiet" mode
	Quiet bool
}

func main() {
	logger := logrus.New()

	// Create a options struct to hold the parsed environment variables or command line flags
	opts := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "NVIDIA Container Toolkit CLI"
	c.UseShortOptionHandling = true
	c.EnableBashCompletion = true
	c.Usage = "Tools to configure the NVIDIA Container Toolkit"
	c.Version = info.GetVersionString()

	// Setup the flags for this command
	c.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Usage:       "Enable debug-level logging",
			Destination: &opts.Debug,
			EnvVars:     []string{"NVIDIA_CTK_DEBUG"},
		},
		&cli.BoolFlag{
			Name:        "quiet",
			Usage:       "Suppress all output except for errors; overrides --debug",
			Destination: &opts.Quiet,
			EnvVars:     []string{"NVIDIA_CTK_QUIET"},
		},
	}

	// Set log-level for all subcommands
	c.Before = func(c *cli.Context) error {
		logLevel := logrus.InfoLevel
		if opts.Debug {
			logLevel = logrus.DebugLevel
		}
		if opts.Quiet {
			logLevel = logrus.ErrorLevel
		}
		logger.SetLevel(logLevel)
		return nil
	}

	// Define the subcommands
	c.Commands = []*cli.Command{
		hook.NewCommand(logger),
		runtime.NewCommand(logger),
		infoCLI.NewCommand(logger),
		cdi.NewCommand(logger),
		system.NewCommand(logger),
		config.NewCommand(logger),
	}

	// Run the CLI
	err := c.Run(os.Args)
	if err != nil {
		logger.Errorf("%v", err)
		os.Exit(1)
	}
}
