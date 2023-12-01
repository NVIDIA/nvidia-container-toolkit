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
	"io"
	"os"

	"github.com/urfave/cli/v2"
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	transformroot "github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform/root"
)

type command struct {
	logger logger.Interface
}

type transformOptions struct {
	input  string
	output string
}

type options struct {
	transformOptions
	from       string
	to         string
	relativeTo string
}

// NewCommand constructs a generate-cdi command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	opts := options{}

	c := cli.Command{
		Name:  "root",
		Usage: "Apply a root transform to a CDI specification",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &opts)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "from",
			Usage:       "specify the root to be transformed",
			Destination: &opts.from,
		},
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
		&cli.StringFlag{
			Name:        "relative-to",
			Usage:       "specify whether the transform is relative to the host or to the container. One of [ host | container ]",
			Value:       "host",
			Destination: &opts.relativeTo,
		},
		&cli.StringFlag{
			Name:        "to",
			Usage:       "specify the replacement root. If this is the same as the from root, the transform is a no-op.",
			Value:       "",
			Destination: &opts.to,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	switch opts.relativeTo {
	case "host":
	case "container":
	default:
		return fmt.Errorf("invalid --relative-to value: %v", opts.relativeTo)
	}
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	spec, err := opts.Load()
	if err != nil {
		return fmt.Errorf("failed to load CDI specification: %w", err)
	}

	err = transformroot.New(
		transformroot.WithRoot(opts.from),
		transformroot.WithTargetRoot(opts.to),
		transformroot.WithRelativeTo(opts.relativeTo),
	).Transform(spec.Raw())
	if err != nil {
		return fmt.Errorf("failed to transform CDI specification: %w", err)
	}

	return opts.Save(spec)
}

// Load lodas the input CDI specification
func (o transformOptions) Load() (spec.Interface, error) {
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

func (o transformOptions) getContents() ([]byte, error) {
	if o.input == "-" {
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(o.input)
}

// Save saves the CDI specification to the output file
func (o transformOptions) Save(s spec.Interface) error {
	if o.output == "" {
		_, err := s.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	return s.Save(o.output)
}
