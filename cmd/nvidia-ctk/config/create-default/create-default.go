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

	nvctkConfig "github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger logger.Interface
}

// options stores the subcommand options
type options struct {
	output string
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
			Name:        "output",
			Usage:       "Specify the file to output the generated configuration for to. If this is '' the configuration is ouput to STDOUT.",
			Destination: &opts.output,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	defaultConfig, err := nvctkConfig.GetDefaultConfigToml()
	if err != nil {
		return fmt.Errorf("unable to get default config: %v", err)
	}

	var output io.Writer
	if opts.output == "" {
		output = os.Stdout
	} else {
		outputFile, err := os.Create(opts.output)
		if err != nil {
			return fmt.Errorf("unable to create output file: %v", err)
		}
		defer outputFile.Close()
		output = outputFile
	}

	_, err = defaultConfig.WriteTo(output)
	if err != nil {
		return fmt.Errorf("unable to write to output: %v", err)
	}

	return nil
}
