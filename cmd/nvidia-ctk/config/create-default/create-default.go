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

package createdefault

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config/flags"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs a default command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	opts := flags.Options{}

	// Create the 'default' command
	c := cli.Command{
		Name:    "default",
		Aliases: []string{"create-default", "generate-default"},
		Usage:   "Generate the default NVIDIA Container Toolkit configuration file",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(cmd, &opts)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.run(cmd, &opts)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Specify the output file to write to; If not specified, the output is written to stdout",
				Destination: &opts.Output,
			},
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Command, opts *flags.Options) error {
	return nil
}

func (m command) run(c *cli.Command, opts *flags.Options) error {
	cfgToml, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to create default config: %v", err)
	}

	output, err := opts.CreateOutput()
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer output.Close()

	if _, err := cfgToml.Save(output); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	return nil
}
