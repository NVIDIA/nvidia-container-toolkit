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

package dotsosymlinks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type command struct {
	logger logger.Interface
}

type config struct {
	containerSpec string
	driverVersion string
}

// NewCommand constructs a hook command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	cfg := config{}

	// Create the '' command
	c := cli.Command{
		Name:  "create-dot-so-symlinks",
		Usage: "A hook to create .so symlinks in the container.",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
		&cli.StringFlag{
			Name:        "driver-version",
			Usage:       "specify the driver version for which the symlinks are to be created. This assumes driver libraries have the .so.`VERSION` suffix.",
			Destination: &cfg.driverVersion,
			Required:    true,
		},
	}

	return &c
}

func (m command) run(c *cli.Context, cfg *config) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRoot, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	locator := lookup.NewLibraryLocator(
		lookup.WithLogger(m.logger),
		lookup.WithRoot(containerRoot),
		lookup.WithOptional(true),
	)
	libs, err := locator.Locate("*.so." + cfg.driverVersion)
	if err != nil {
		return fmt.Errorf("failed to locate libraries for driver version %v: %v", cfg.driverVersion, err)
	}

	for _, lib := range libs {
		if !strings.HasSuffix(lib, ".so."+cfg.driverVersion) {
			continue
		}
		libSoPath := strings.TrimSuffix(lib, "."+cfg.driverVersion)
		libSoXPaths, err := filepath.Glob(libSoPath + ".[0-9]")
		if len(libSoXPaths) != 1 || err != nil {
			continue
		}
		err = os.Symlink(filepath.Base(libSoXPaths[0]), libSoPath)
		if err != nil {
			continue
		}
	}
	return nil
}
