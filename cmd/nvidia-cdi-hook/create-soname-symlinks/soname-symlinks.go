/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package soname

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	safeexec "github.com/NVIDIA/nvidia-container-toolkit/internal/safe-exec"
)

type command struct {
	logger logger.Interface
	safeexec.Execer
}

type options struct {
	folders       cli.StringSlice
	ldconfigPath  string
	containerSpec string
}

// NewCommand constructs an create-soname-symlinks command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
		Execer: safeexec.New(logger),
	}
	return c.build()
}

// build the create-soname-symlinks command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'create-soname-symlinks' command
	c := cli.Command{
		Name:  "create-soname-symlinks",
		Usage: "Create soname symlinks for the specified folders using ldconfig -n -N",
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
			Usage:       "Specify a folder to search for shared libraries for which soname symlinks need to be created",
			Destination: &cfg.folders,
		},
		&cli.StringFlag{
			Name:        "ldconfig-path",
			Usage:       "Specify the path to the ldconfig program",
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

	containerRoot, err := s.GetContainerRootDirPath()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %v", err)
	}
	if containerRoot == "" {
		m.logger.Warningf("No container root detected")
		return nil
	}

	dirs := cfg.folders.Value()
	if len(dirs) == 0 {
		return nil
	}

	ldconfigPath := config.ResolveLDConfigPathOnHost(cfg.ldconfigPath)
	args := []string{filepath.Base(ldconfigPath)}

	args = append(args,
		// Specify the containerRoot to use.
		"-r", string(containerRoot),
		// Specify -n to only process the specified folders.
		"-n",
		// Explicitly disable updating the LDCache.
		"-N",
	)
	// Explicitly specific the directories to add.
	args = append(args, dirs...)

	return m.Exec(ldconfigPath, args, nil)
}
