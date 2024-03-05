/**
# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

package list

import (
	"encoding/json"
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/runtime"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs an list command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// options defines the options that can be set for the CLI through options files,
type options struct {
	mode    string
	envvars cli.StringSlice
}

func (m command) build() *cli.Command {
	// Create a options struct to hold the parsed environment variables or command line flags
	cfg := options{}

	// Create the 'configure' command
	configure := cli.Command{
		Name:  "list",
		Usage: "List the modifications made to the OCI runtime specification",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.list(c, &cfg)
		},
	}

	configure.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "mode",
			Usage:       "override the runtime mode a specified in the config file",
			Destination: &cfg.mode,
		},
		&cli.StringSliceFlag{
			Name:        "envvar",
			Aliases:     []string{"e"},
			Usage:       "add an envvar to the container definition",
			Destination: &cfg.envvars,
		},
	}

	return &configure
}

func (m command) validateFlags(c *cli.Context, cfg *options) error {
	return nil
}

// list executes the list command.
func (m command) list(c *cli.Context, cfg *options) error {
	toolkitConfig, err := config.GetDefault()
	if err != nil {
		return fmt.Errorf("failed to generate default config: %w", err)
	}

	if cfg.mode != "" {
		toolkitConfig.NVIDIAContainerRuntimeConfig.Mode = cfg.mode
	}

	container, err := image.New(
		image.WithEnv(cfg.envvars.Value()),
	)
	if err != nil {
		return fmt.Errorf("failed to construct container image: %w", err)
	}

	modifier, err := runtime.NewSpecModifier(m.logger, toolkitConfig, container)
	if err != nil {
		return fmt.Errorf("failed to contruct OCI runtime specification modifier: %w", err)
	}
	// TODO: We should handle this more cleanly.
	if modifier == nil {
		return fmt.Errorf("no modifications required")
	}

	spec := specs.Spec{}
	if err := modifier.Modify(&spec); err != nil {
		return fmt.Errorf("faile to apply modification to empty spec: %w", err)
	}

	specJSON, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OCI spec to JSON: %w", err)
	}
	fmt.Printf("%v", string(specJSON))

	return nil
}
