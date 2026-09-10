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

package ldcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/moby/sys/reexec"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldconfig"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	reexecUpdateLdCacheCommandName = "reexec-update-ldcache"
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
	reexec.Register(reexecUpdateLdCacheCommandName, updateLdCacheHandler)
	if reexec.Init() {
		os.Exit(0)
	}
}

// NewCommand constructs an update-ldcache command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the update-ldcache command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'update-ldcache' command
	c := cli.Command{
		Name:  "update-ldcache",
		Usage: "Update ldcache in a container by running ldconfig",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(cmd, &cfg)
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return m.run(cmd, &cfg)
		},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "folder",
				Usage:       "Specify a folder to add to /etc/ld.so.conf before updating the ld cache",
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

	runner, err := ldconfig.NewRunner(
		reexecUpdateLdCacheCommandName,
		cfg.ldconfigPath,
		containerRootDir,
		m.compat32Requested(s),
		cfg.folders...,
	)
	if err != nil {
		return err
	}
	return runner.Run()
}

// compat32Requested checks whether the container requested the compat32
// driver capability. This mirrors the nvidia-container-cli, where the 32-bit
// libraries are associated with this capability.
// Note that the environment of the container is read from its OCI spec, which
// is not always accessible, and a container whose capabilities cannot be
// determined is treated as not having requested this one.
func (m command) compat32Requested(s *oci.State) bool {
	env, err := s.GetEnv()
	if err != nil {
		m.logger.Debugf("Failed to get the container environment: %v", err)
		return false
	}
	containerImage, err := image.New(image.WithEnv(env))
	if err != nil {
		m.logger.Debugf("Failed to construct image from the container environment: %v", err)
		return false
	}
	return containerImage.GetDriverCapabilities().Has(image.DriverCapabilityCompat32)
}

// updateLdCacheHandler wraps updateLdCache with error handling.
func updateLdCacheHandler() {
	if err := updateLdCache(os.Args); err != nil {
		log.Fatalf("Error updating ldcache: %v", err)
	}
}

// updateLdCache ensures that the ldcache in the container is updated to include
// libraries that are mounted from the host.
// It is invoked from a reexec'd handler and provides namespace isolation for
// the operations performed by this hook. At the point where this is invoked,
// we are in a new mount namespace that is cloned from the parent.
func updateLdCache(args []string) error {
	ldconfig, err := ldconfig.NewFromArgs(args...)
	if err != nil {
		return fmt.Errorf("failed to construct ldconfig runner: %w", err)
	}

	return ldconfig.UpdateLDCache()
}
