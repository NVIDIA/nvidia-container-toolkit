/**
# Copyright 2024 NVIDIA CORPORATION
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

package resolve

import (
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs an resolve command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

type config struct {
	mode string
}

func (m command) build() *cli.Command {
	// Create a config struct to hold the parsed environment variables or command line flags
	config := config{}

	cmd := cli.Command{
		Name:  "resolve",
		Usage: "Add a runtime to the specified container engine",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &config)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &config)
		},
	}

	cmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "mode",
			Value:       "auto",
			Destination: &config.mode,
		},
	}

	return &cmd
}

func (m command) validateFlags(c *cli.Context, config *config) error {
	return nil
}

func (m command) run(c *cli.Context, config *config) error {
	infolib := info.New(
		info.WithLogger(m.logger),
	)

	mode := infolib.Resolve(config.mode)
	m.logger.Infof("Resolved mode as %q", mode)
	return nil
}
