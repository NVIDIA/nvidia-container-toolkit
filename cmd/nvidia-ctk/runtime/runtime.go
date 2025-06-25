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
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/runtime/configure"
)

type command struct {
	logger     *logrus.Logger
	configFile *string
}

// NewCommand constructs a runtime command with the specified logger
func NewCommand(logger *logrus.Logger, configFile *string) *cli.Command {
	c := command{
		logger:     logger,
		configFile: configFile,
	}
	return c.build()
}

func (m command) build() *cli.Command {
	// Create the 'runtime' command
	runtime := cli.Command{
		Name:  "runtime",
		Usage: "A collection of runtime-related utilities for the NVIDIA Container Toolkit",
	}

	runtime.Commands = []*cli.Command{
		configure.NewCommand(m.logger),
	}

	return &runtime
}
