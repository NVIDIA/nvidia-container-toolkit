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

package defaultsubcommand

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger logger.Interface
}

// options stores the subcommand options
type options struct {
	config  string
	output  string
	inPlace bool
}

// NewCommand constructs a default command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	opts := options{}

	// Create the 'default' command
	c := cli.Command{
		Name:    "generate-default",
		Aliases: []string{"default"},
		Usage:   "Generate the default NVIDIA Container Toolkit configuration file",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &opts)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Specify the config file to process; The contents of this file overrides the default config",
			Destination: &opts.config,
		},
		&cli.BoolFlag{
			Name:        "in-place",
			Aliases:     []string{"i"},
			Usage:       "Modify the config file in-place",
			Destination: &opts.inPlace,
		},
		&cli.StringFlag{
			Name:        "output",
			Usage:       "Specify the output file to write to; If not specified, the output is written to stdout",
			Destination: &opts.output,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	if opts.inPlace {
		if opts.output != "" {
			return fmt.Errorf("cannot specify both --in-place and --output")
		}
		opts.output = opts.config
	}
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	if err := opts.ensureOutputFolder(); err != nil {
		return fmt.Errorf("unable to create output directory: %v", err)
	}

	cfgToml, err := opts.getConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	if _, err := opts.Write(cfgToml); err != nil {
		return fmt.Errorf("unable to write to output: %v", err)
	}

	return nil
}

// getConfig returns the TOML config for the specified options.
func (opts options) getConfig() (*config.Toml, error) {
	return config.New(
		config.WithConfigFile(opts.config),
	)
}

func (opts options) ensureOutputFolder() error {
	if opts.output == "" {
		return nil
	}
	if dir := filepath.Dir(opts.output); dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// Write writes the contents to the output file specified in the options.
func (opts options) Write(cfg *config.Toml) (int, error) {
	var output io.Writer
	if opts.output == "" {
		output = os.Stdout
	} else {
		outputFile, err := os.Create(opts.output)
		if err != nil {
			return 0, fmt.Errorf("unable to create output file: %v", err)
		}
		defer outputFile.Close()
		output = outputFile
	}

	n, err := cfg.Save(output)
	return int(n), err
}
