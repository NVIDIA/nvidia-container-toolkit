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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvdevices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvmodules"
)

type command struct {
	logger logger.Interface
}

type options struct {
	driverRoot string

	dryRun bool

	control bool

	loadKernelModules bool
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
		Name:  "create-device-nodes",
		Usage: "A utility to create NVIDIA device nodes",
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
		&cli.BoolFlag{
			Name:        "control-devices",
			Usage:       "create all control device nodes: nvidiactl, nvidia-modeset, nvidia-uvm, nvidia-uvm-tools",
			Destination: &opts.control,
		},
		&cli.BoolFlag{
			Name:        "load-kernel-modules",
			Usage:       "load the NVIDIA Kernel Modules before creating devices nodes",
			Destination: &opts.loadKernelModules,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "if set, the command will not create any symlinks.",
			Value:       false,
			Destination: &opts.dryRun,
			EnvVars:     []string{"DRY_RUN"},
		},
	}

	return &c
}

func (m command) validateFlags(r *cli.Context, opts *options) error {
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	if opts.loadKernelModules {
		modules := nvmodules.New(
			nvmodules.WithLogger(m.logger),
			nvmodules.WithDryRun(opts.dryRun),
			nvmodules.WithRoot(opts.driverRoot),
		)
		if err := modules.LoadAll(); err != nil {
			return fmt.Errorf("failed to load NVIDIA kernel modules: %v", err)
		}
	}

	if opts.control {
		devices, err := nvdevices.New(
			nvdevices.WithLogger(m.logger),
			nvdevices.WithDryRun(opts.dryRun),
			nvdevices.WithDevRoot(opts.driverRoot),
		)
		if err != nil {
			return err
		}
		m.logger.Infof("Creating control device nodes at %s", opts.driverRoot)
		if err := devices.CreateNVIDIAControlDevices(); err != nil {
			return fmt.Errorf("failed to create NVIDIA control device nodes: %v", err)
		}
	}
	return nil
}
