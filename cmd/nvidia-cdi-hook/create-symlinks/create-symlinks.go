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

package symlinks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type command struct {
	logger logger.Interface
}

type config struct {
	hostRoot      string
	links         cli.StringSlice
	containerSpec string
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
		Name:  "create-symlinks",
		Usage: "A hook to create symlinks in the container.",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "The root on the host filesystem to use to resolve symlinks",
			Destination: &cfg.hostRoot,
		},
		&cli.StringSliceFlag{
			Name:        "link",
			Usage:       "Specify a specific link to create. The link is specified as target::link",
			Destination: &cfg.links,
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
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

	created := make(map[string]bool)
	for _, l := range cfg.links.Value() {
		parts := strings.Split(l, "::")
		if len(parts) != 2 {
			m.logger.Warningf("Invalid link specification %v", l)
			continue
		}

		err := m.createLink(created, cfg.hostRoot, containerRoot, parts[0], parts[1])
		if err != nil {
			m.logger.Warningf("Failed to create link %v: %v", parts, err)
		}
	}
	return nil
}

func (m command) createLink(created map[string]bool, hostRoot string, containerRoot string, target string, link string) error {
	linkPath, err := changeRoot(hostRoot, containerRoot, link)
	if err != nil {
		m.logger.Warningf("Failed to resolve path for link %v relative to %v: %v", link, containerRoot, err)
	}
	if created[linkPath] {
		m.logger.Debugf("Link %v already created", linkPath)
		return nil
	}

	targetPath, err := changeRoot(hostRoot, "/", target)
	if err != nil {
		m.logger.Warningf("Failed to resolve path for target %v relative to %v: %v", target, "/", err)
	}

	m.logger.Infof("Symlinking %v to %v", linkPath, targetPath)
	err = os.MkdirAll(filepath.Dir(linkPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	err = os.Symlink(target, linkPath)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}

	return nil
}

func changeRoot(current string, new string, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}

	relative := path
	if current != "" {
		r, err := filepath.Rel(current, path)
		if err != nil {
			return "", err
		}
		relative = r
	}

	return filepath.Join(new, relative), nil
}
