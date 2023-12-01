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

package createdevicenodes

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type command struct {
	logger logger.Interface
}

type options struct {
	driverRoot string
}

// NewCommand constructs a command sub-command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	opts := options{}

	c := cli.Command{
		Name:  "print-ldcache",
		Usage: "A utility to print the contents of the ldcache",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &opts)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "driver-root",
			Usage:       "the path to the driver root. Device nodes will be created at `DRIVER_ROOT`/dev",
			Value:       "/",
			Destination: &opts.driverRoot,
			EnvVars:     []string{"DRIVER_ROOT"},
		},
	}

	return &c
}

func (m command) validateFlags(r *cli.Context, opts *options) error {
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	cache, err := ldcache.New(m.logger, opts.driverRoot)
	if err != nil {
		return fmt.Errorf("failed to create ldcache: %v", err)
	}

	lib32, lib64 := cache.List()

	if len(lib32) == 0 {
		m.logger.Info("No 32-bit libraries found")
	} else {
		m.logger.Infof("%d 32-bit libraries found", len(lib32))
		for _, lib := range lib32 {
			m.logger.Infof("%v", lib)
		}
	}
	if len(lib64) == 0 {
		m.logger.Info("No 64-bit libraries found")
	} else {
		m.logger.Infof("%d 64-bit libraries found", len(lib64))
		for _, lib := range lib64 {
			m.logger.Infof("%v", lib)
		}
	}

	return nil
}
