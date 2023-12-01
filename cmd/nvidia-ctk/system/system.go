/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package system

import (
	"github.com/urfave/cli/v2"

	devchar "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/system/create-dev-char-symlinks"
	devicenodes "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/system/create-device-nodes"
	ldcache "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/system/print-ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger logger.Interface
}

// NewCommand constructs a runtime command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

func (m command) build() *cli.Command {
	// Create the 'system' command
	system := cli.Command{
		Name:  "system",
		Usage: "A collection of system-related utilities for the NVIDIA Container Toolkit",
	}

	system.Subcommands = []*cli.Command{
		devchar.NewCommand(m.logger),
		devicenodes.NewCommand(m.logger),
		ldcache.NewCommand(m.logger),
	}

	return &system
}
