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

package cdi

import (
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi/generate"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi/list"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi/transform"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger         logger.Interface
	configFilePath *string
}

// NewCommand constructs a cdi command with the specified logger
func NewCommand(logger logger.Interface, configFilePath *string) *cli.Command {
	c := command{
		logger:         logger,
		configFilePath: configFilePath,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	// Create the 'cdi' command
	cdi := cli.Command{
		Name:  "cdi",
		Usage: "Provide tools for interacting with Container Device Interface specifications",
		Commands: []*cli.Command{
			generate.NewCommand(m.logger, m.configFilePath),
			list.NewCommand(m.logger),
			transform.NewCommand(m.logger),
		},
	}

	return &cdi
}
