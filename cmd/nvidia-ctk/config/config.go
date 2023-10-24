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

package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	createdefault "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config/create-default"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk/config/flags"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger logger.Interface
}

// options stores the subcommand options
type options struct {
	flags.Options
	sets cli.StringSlice
}

// NewCommand constructs an config command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	opts := options{}

	// Create the 'config' command
	c := cli.Command{
		Name:  "config",
		Usage: "Interact with the NVIDIA Container Toolkit configuration",
		Action: func(ctx *cli.Context) error {
			return run(ctx, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"config", "c"},
			Usage:       "Specify the config file to modify.",
			Value:       config.GetConfigFilePath(),
			Destination: &opts.Config,
		},
		&cli.StringSliceFlag{
			Name:        "set",
			Usage:       "Set a config value using the pattern key=value. If value is empty, this is equivalent to specifying the same key in unset. This flag can be specified multiple times",
			Destination: &opts.sets,
		},
		&cli.BoolFlag{
			Name:        "in-place",
			Aliases:     []string{"i"},
			Usage:       "Modify the config file in-place",
			Destination: &opts.InPlace,
		},
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Usage:       "Specify the output file to write to; If not specified, the output is written to stdout",
			Destination: &opts.Output,
		},
	}

	c.Subcommands = []*cli.Command{
		createdefault.NewCommand(m.logger),
	}

	return &c
}

func run(c *cli.Context, opts *options) error {
	cfgToml, err := config.New(
		config.WithConfigFile(opts.Config),
	)
	if err != nil {
		return fmt.Errorf("unable to create config: %v", err)
	}

	for _, set := range opts.sets.Value() {
		key, value, err := (*configToml)(cfgToml).setFlagToKeyValue(set)
		if err != nil {
			return fmt.Errorf("invalid --set option %v: %w", set, err)
		}
		cfgToml.Set(key, value)
	}

	if err := opts.EnsureOutputFolder(); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	output, err := opts.CreateOutput()
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer output.Close()

	if _, err := cfgToml.Save(output); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	return nil
}

type configToml config.Toml

var errInvalidConfigOption = errors.New("invalid config option")
var errInvalidFormat = errors.New("invalid format")

// setFlagToKeyValue converts a --set flag to a key-value pair.
// The set flag is of the form key[=value], with the value being optional if key refers to a
// boolean config option.
func (c *configToml) setFlagToKeyValue(setFlag string) (string, interface{}, error) {
	if c == nil {
		return "", nil, errInvalidConfigOption
	}

	setParts := strings.SplitN(setFlag, "=", 2)
	key := setParts[0]

	v := (*config.Toml)(c).Get(key)
	if v == nil {
		return key, nil, errInvalidConfigOption
	}
	if _, ok := v.(bool); ok {
		if len(setParts) == 1 {
			return key, true, nil
		}
	}

	if len(setParts) != 2 {
		return key, nil, fmt.Errorf("%w: expected key=value; got %v", errInvalidFormat, setFlag)
	}

	value := setParts[1]
	switch vt := v.(type) {
	case bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return key, value, fmt.Errorf("%w: %w", errInvalidFormat, err)
		}
		return key, b, err
	case string:
		return key, value, nil
	case []string:
		return key, strings.Split(value, ","), nil
	default:
		return key, nil, fmt.Errorf("unsupported type for %v (%v)", setParts, vt)
	}
}
