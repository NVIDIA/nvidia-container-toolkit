/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package create_soname_symlinks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/moby/sys/reexec"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldconfig"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	reexecUpdateLdCacheCommandName = "reexec-create-soname-symlinks"
)

type command struct {
	logger logger.Interface
}

type options struct {
	folders       []string
	ldconfigPath  string
	containerSpec string
}

func init() {
	reexec.Register(reexecUpdateLdCacheCommandName, createSonameSymlinksHandler)
	if reexec.Init() {
		os.Exit(0)
	}
}

// NewCommand constructs an create-soname-symlinks command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the create-soname-symlinks command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'create-soname-symlinks' command
	c := cli.Command{
		Name:  "create-soname-symlinks",
		Usage: "Create soname symlinks libraries in specified directories",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(cmd, &cfg)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.run(cmd, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "folder",
			Usage:       "Specify a directory to generate soname symlinks in. Can be specified multiple times",
			Destination: &cfg.folders,
		},
		&cli.StringFlag{
			Name:        "ldconfig-path",
			Usage:       "Specify the path to ldconfig on the host",
			Destination: &cfg.ldconfigPath,
			Value:       "/sbin/ldconfig",
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) validateFlags(_ *cli.Command, cfg *options) error {
	if cfg.ldconfigPath == "" {
		return errors.New("ldconfig-path must be specified")
	}
	return nil
}

func (m command) run(_ *cli.Command, cfg *options) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRootDir, err := s.GetContainerRoot()
	if err != nil || containerRootDir == "" || containerRootDir == "/" {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	cmd, err := ldconfig.NewRunner(
		reexecUpdateLdCacheCommandName,
		cfg.ldconfigPath,
		containerRootDir,
		cfg.folders...,
	)
	if err != nil {
		return err
	}

	return cmd.Run()
}

// createSonameSymlinksHandler wraps createSonameSymlinks with error handling.
func createSonameSymlinksHandler() {
	if err := createSonameSymlinks(os.Args); err != nil {
		log.Printf("Error updating ldcache: %v", err)
		os.Exit(1)
	}
}

// createSonameSymlinks ensures that soname symlinks are created in the
// specified directories.
// It is invoked from a reexec'd handler and provides namespace isolation for
// the operations performed by this hook. At the point where this is invoked,
// we are in a new mount namespace that is cloned from the parent.
//
// args[0] is the reexec initializer function name
// args[1] is the path of the ldconfig binary on the host
// args[2] is the container root directory
// The remaining args are directories where soname symlinks need to be created.
func createSonameSymlinks(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("incorrect arguments: %v", args)
	}
	hostLdconfigPath := args[1]
	containerRootDirPath := args[2]

	ldconfig, err := ldconfig.New(
		hostLdconfigPath,
		containerRootDirPath,
	)
	if err != nil {
		return fmt.Errorf("failed to construct ldconfig runner: %w", err)
	}

	return ldconfig.CreateSonameSymlinks(args[3:]...)
}
