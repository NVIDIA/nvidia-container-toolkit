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
	"fmt"
	"io"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/cdi/transform/root"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger *logrus.Logger
}

type options struct {
	input  string
	output string
}

// NewCommand constructs a command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	opts := options{}

	c := cli.Command{
		Name:  "transform",
		Usage: "Apply a transform to a CDI specification",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &opts)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "input",
			Usage:       "Specify the file to read the CDI specification from. If this is '-' the specification is read from STDIN",
			Value:       "-",
			Destination: &opts.input,
		},
		&cli.StringFlag{
			Name:        "output",
			Usage:       "Specify the file to output the generated CDI specification to. If this is '' the specification is output to STDOUT",
			Destination: &opts.output,
		},
	}

	c.Subcommands = []*cli.Command{
		root.NewCommand(m.logger, &opts),
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	return nil
}

func (m command) run(c *cli.Context, cfg *options) error {
	return nil
}

// Load lodas the input CDI specification
func (o options) Load() (spec.Interface, error) {
	contents, err := o.getContents()
	if err != nil {
		return nil, fmt.Errorf("failed to read spec contents: %v", err)
	}

	raw, err := cdi.ParseSpec(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CDI spec: %v", err)
	}

	return spec.New(
		spec.WithRawSpec(raw),
	)
}

func (o options) getContents() ([]byte, error) {
	if o.input == "-" {
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(o.input)
}

// Save saves the CDI specification to the output file
func (o options) Save(s spec.Interface) error {
	if o.output == "" {
		_, err := s.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	return s.Save(o.output)
}
