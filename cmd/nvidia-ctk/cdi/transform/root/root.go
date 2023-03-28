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

package root

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type loadSaver interface {
	Load() (spec.Interface, error)
	Save(spec.Interface) error
}

type command struct {
	logger *logrus.Logger

	handler loadSaver
}

type config struct {
	from string
	to   string
}

// NewCommand constructs a generate-cdi command with the specified logger
func NewCommand(logger *logrus.Logger, specHandler loadSaver) *cli.Command {
	c := command{
		logger:  logger,
		handler: specHandler,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	cfg := config{}

	c := cli.Command{
		Name:  "root",
		Usage: "Apply a root transform to a CDI specification",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "from",
			Usage:       "specify the root to be transformed",
			Destination: &cfg.from,
		},
		&cli.StringFlag{
			Name:        "to",
			Usage:       "specify the replacement root. If this is the same as the from root, the transform is a no-op.",
			Value:       "",
			Destination: &cfg.to,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, cfg *config) error {
	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.handler.Load()
	if err != nil {
		return fmt.Errorf("failed to load CDI specification: %w", err)
	}

	err = transform.NewRootTransformer(
		cfg.from,
		cfg.to,
	).Transform(spec.Raw())
	if err != nil {
		return fmt.Errorf("failed to transform CDI specification: %w", err)
	}

	return m.handler.Save(spec)
}
