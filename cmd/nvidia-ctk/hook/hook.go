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
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/commands"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

	"github.com/urfave/cli/v2"
)

type hookCommand struct {
	logger logger.Interface
}

// NewCommand constructs CLI subcommand for handling CDI hooks.
func NewCommand(logger logger.Interface) *cli.Command {
	c := hookCommand{
		logger: logger,
	}
	return c.build()
}

// build
func (m hookCommand) build() *cli.Command {
	// Create the 'hook' subcommand
	hook := cli.Command{
		Name:  "hook",
		Usage: "A collection of hooks that may be injected into an OCI spec",
		// We set the default action for the `hook` subcommand to issue a
		// warning and exit with no error.
		// This means that if an unsupported hook is run, a container will not fail
		// to launch. An unsupported hook could be the result of a CDI specification
		// referring to a new hook that is not yet supported by an older NVIDIA
		// Container Toolkit version or a hook that has been removed in newer
		// version.
		Action: func(ctx *cli.Context) error {
			commands.IssueUnsupportedHookWarning(m.logger, ctx)
			return nil
		},
	}

	hook.Subcommands = commands.New(m.logger)

	return &hook
}
