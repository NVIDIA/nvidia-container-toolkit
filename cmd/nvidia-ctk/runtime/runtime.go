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

package runtime

import (
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/runtime/configure"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type runtimeCommand struct {
	logger logger.Interface
}

// NewCommand constructs a runtime command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := runtimeCommand{
		logger: logger,
	}
	return c.build()
}

func (m runtimeCommand) build() *cli.Command {
	// Create the 'runtime' command
	runtime := cli.Command{
		Name:  "runtime",
		Usage: "A collection of runtime-related utilities for the NVIDIA Container Toolkit",
	}

	runtime.Subcommands = []*cli.Command{
		configure.NewCommand(m.logger),
	}

	return &runtime
}
