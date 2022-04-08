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

package hook

import (
	symlinks "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/hook/create-symlinks"
	ldcache "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/hook/update-ldcache"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type hookCommand struct {
	logger *logrus.Logger
}

// NewCommand constructs a hook command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := hookCommand{
		logger: logger,
	}
	return c.build()
}

// build
func (m hookCommand) build() *cli.Command {
	// Create the 'hook' command
	hook := cli.Command{
		Name:  "hook",
		Usage: "A collection of hooks that may be injected into an OCI spec",
	}

	hook.Subcommands = []*cli.Command{
		ldcache.NewCommand(m.logger),
		symlinks.NewCommand(m.logger),
	}

	return &hook
}
