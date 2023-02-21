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

package chmod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type command struct {
	logger *logrus.Logger
}

type config struct {
	paths         cli.StringSlice
	mode          string
	containerSpec string
}

// NewCommand constructs a chmod command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the chmod command
func (m command) build() *cli.Command {
	cfg := config{}

	// Create the 'chmod' command
	c := cli.Command{
		Name:  "chmod",
		Usage: "Set the permissions of folders in the container by running chmod. The container root is prefixed to the specified paths.",
		Before: func(c *cli.Context) error {
			return validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "path",
			Usage:       "Specifiy a path to apply the specified mode to",
			Destination: &cfg.paths,
		},
		&cli.StringFlag{
			Name:        "mode",
			Usage:       "Specify the file mode",
			Destination: &cfg.mode,
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func validateFlags(c *cli.Context, cfg *config) error {
	if strings.TrimSpace(cfg.mode) == "" {
		return fmt.Errorf("a non-empty mode must be specified")
	}

	for _, p := range cfg.paths.Value() {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("paths must not be empty")
		}
	}

	return nil
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
	if containerRoot == "" {
		return fmt.Errorf("empty container root detected")
	}

	paths := m.getPaths(containerRoot, cfg.paths.Value())
	if len(paths) == 0 {
		m.logger.Debugf("No paths specified; exiting")
		return nil
	}

	locator := lookup.NewExecutableLocator(m.logger, "")
	targets, err := locator.Locate("chmod")
	if err != nil {
		return fmt.Errorf("failed to locate chmod: %v", err)
	}
	chmodPath := targets[0]

	args := append([]string{filepath.Base(chmodPath), cfg.mode}, paths...)

	return syscall.Exec(chmodPath, args, nil)
}

// getPaths updates the specified paths relative to the root.
func (m command) getPaths(root string, paths []string) []string {
	var pathsInRoot []string
	for _, f := range paths {
		path := filepath.Join(root, f)
		if _, err := os.Stat(path); err != nil {
			m.logger.Debugf("Skipping path %q: %v", path, err)
			continue
		}
		pathsInRoot = append(pathsInRoot, path)
	}

	return pathsInRoot
}
