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
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/moby/sys/reexec"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
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
	folders       cli.StringSlice
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
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
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

func (m command) validateFlags(c *cli.Context, cfg *options) error {
	if cfg.ldconfigPath == "" {
		return errors.New("ldconfig-path must be specified")
	}
	return nil
}

func (m command) run(c *cli.Context, cfg *options) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRootDir, err := s.GetContainerRoot()
	if err != nil || containerRootDir == "" || containerRootDir == "/" {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	args := []string{
		reexecUpdateLdCacheCommandName,
		strings.TrimPrefix(config.NormalizeLDConfigPath("@"+cfg.ldconfigPath), "@"),
		containerRootDir,
	}
	args = append(args, cfg.folders.Value()...)

	cmd := createReexecCommand(args)

	return cmd.Run()
}

// createSonameSymlinksHandler wraps createSonameSymlinks with error handling.
func createSonameSymlinksHandler() {
	if err := createSonameSymlinks(os.Args); err != nil {
		log.Printf("Error updating ldcache: %v", err)
		os.Exit(1)
	}
}

// createSonameSymlinks is invoked from a reexec'd handler and provides namespace
// isolation for the operations performed by this hook.
// At the point where this is invoked, we are in a new mount namespace that is
// cloned from the parent.
//
// args[0] is the reexec initializer function name
// args[1] is the path of the ldconfig binary on the host
// args[2] is the container root directory
// The remaining args are directories that need to be added to the ldcache.
func createSonameSymlinks(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("incorrect arguments: %v", args)
	}
	hostLdconfigPath := args[1]
	containerRootDirPath := args[2]

	// To prevent leaking the parent proc filesystem, we create a new proc mount
	// in the container root.
	if err := mountProc(containerRootDirPath); err != nil {
		return fmt.Errorf("error mounting /proc: %w", err)
	}

	// We mount the host ldconfig before we pivot root since host paths are not
	// visible after the pivot root operation.
	ldconfigPath, err := mountLdConfig(hostLdconfigPath, containerRootDirPath)
	if err != nil {
		return fmt.Errorf("error mounting host ldconfig: %w", err)
	}

	// We pivot to the container root for the new process, this further limits
	// access to the host.
	if err := pivotRoot(containerRootDirPath); err != nil {
		return fmt.Errorf("error running pivot_root: %w", err)
	}

	return runLdconfig(ldconfigPath, args[3:]...)
}

// runLdconfig runs the ldconfig binary and ensures that soname symlinks are
// created in the specified directories.
func runLdconfig(ldconfigPath string, directories ...string) error {
	args := []string{
		"ldconfig",
		// Explicitly disable updating the LDCache.
		"-N",
		// Specify -n to only process the specified directories.
		"-n",
	}
	args = append(args, directories...)

	return SafeExec(ldconfigPath, args, nil)
}
