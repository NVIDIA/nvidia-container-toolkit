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

package config

import (
	createdefault "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config/create-default"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs an config command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	// Create the 'config' command
	c := cli.Command{
		Name:  "config",
		Usage: "Interact with the NVIDIA Container Toolkit configuration",
	}

	c.Subcommands = []*cli.Command{
		createdefault.NewCommand(m.logger),
	}

	return &c
}
