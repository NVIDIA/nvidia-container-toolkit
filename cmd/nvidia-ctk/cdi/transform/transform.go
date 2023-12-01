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

package transform

import (
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi/transform/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs a command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	c := cli.Command{
		Name:  "transform",
		Usage: "Apply a transform to a CDI specification",
	}

	c.Flags = []cli.Flag{}

	c.Subcommands = []*cli.Command{
		root.NewCommand(m.logger),
	}

	return &c
}
