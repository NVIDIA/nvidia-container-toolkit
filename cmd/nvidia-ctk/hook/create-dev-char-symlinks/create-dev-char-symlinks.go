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

package devchar

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	defaultDevCharPath = "/dev/char"
)

type command struct {
	logger *logrus.Logger
}

type config struct {
	devCharPath string
	driverRoot  string
	dryRun      bool
}

// NewCommand constructs a hook sub-command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	cfg := config{}

	// Create the 'create-dev-char-symlinks' command
	c := cli.Command{
		Name:  "create-dev-char-symlinks",
		Usage: "A hook to create symlinks to possible /dev/nv* devices in /dev/char",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "dev-char-path",
			Usage:       "The path at which the symlinks will be created. Symlinks will be created as `DEV_CHAR`/MAJOR:MINOR where MAJOR and MINOR are the major and minor numbers of a corresponding device node.",
			Value:       defaultDevCharPath,
			Destination: &cfg.devCharPath,
			EnvVars:     []string{"DEV_CHAR_PATH"},
		},
		&cli.StringFlag{
			Name:        "driver-root",
			Usage:       "The path to the driver root. `DRIVER_ROOT`/dev is searched for NVIDIA device nodes.",
			Value:       "/",
			Destination: &cfg.driverRoot,
			EnvVars:     []string{"DRIVER_ROOT"},
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "If set, the command will not create any symlinks.",
			Value:       false,
			Destination: &cfg.dryRun,
			EnvVars:     []string{"DRY_RUN"},
		},
	}

	return &c
}

func (m command) run(c *cli.Context, cfg *config) error {
	l := NewSymlinkCreator(
		WithLogger(m.logger),
		WithDevCharPath(cfg.devCharPath),
		WithDriverRoot(cfg.driverRoot),
		WithDryRun(cfg.dryRun),
	)
	err := l.CreateLinks()
	if err != nil {
		return fmt.Errorf("failed to create links: %v", err)
	}
	return nil
}

type linkCreator struct {
	logger      *logrus.Logger
	lister      nodeLister
	driverRoot  string
	devCharPath string
	dryRun      bool
}

// Creator is an interface for creating symlinks to /dev/nv* devices in /dev/char.
type Creator interface {
	CreateLinks() error
}

// Option is a functional option for configuring the linkCreator.
type Option func(*linkCreator)

// NewSymlinkCreator creates a new linkCreator.
func NewSymlinkCreator(opts ...Option) Creator {
	c := linkCreator{}
	for _, opt := range opts {
		opt(&c)
	}
	if c.logger == nil {
		c.logger = logrus.StandardLogger()
	}
	if c.driverRoot == "" {
		c.driverRoot = "/"
	}
	if c.devCharPath == "" {
		c.devCharPath = defaultDevCharPath
	}
	if c.lister == nil {
		c.lister = existing{c.logger, c.driverRoot}
	}
	return c
}

// WithDriverRoot sets the driver root path.
func WithDriverRoot(root string) Option {
	return func(c *linkCreator) {
		c.driverRoot = root
	}
}

// WithDevCharPath sets the path at which the symlinks will be created.
func WithDevCharPath(path string) Option {
	return func(c *linkCreator) {
		c.devCharPath = path
	}
}

// WithDryRun sets the dry run flag.
func WithDryRun(dryRun bool) Option {
	return func(c *linkCreator) {
		c.dryRun = dryRun
	}
}

// WithLogger sets the logger.
func WithLogger(logger *logrus.Logger) Option {
	return func(c *linkCreator) {
		c.logger = logger
	}
}

// CreateLinks creates symlinks for all device nodes returned by the configured lister.
func (m linkCreator) CreateLinks() error {
	deviceNodes, err := m.lister.DeviceNodes()
	if err != nil {
		return fmt.Errorf("failed to get device nodes: %v", err)
	}

	if len(deviceNodes) != 0 && !m.dryRun {
		err := os.MkdirAll(m.devCharPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", m.devCharPath, err)
		}
	}

	for _, deviceNode := range deviceNodes {
		target := deviceNode.path
		linkPath := filepath.Join(m.devCharPath, deviceNode.devCharName())

		m.logger.Infof("Creating link %s => %s", linkPath, target)
		if m.dryRun {
			continue
		}

		err = os.Symlink(target, linkPath)
		if err != nil {
			m.logger.Warnf("Could not create symlink: %v", err)
		}
	}

	return nil
}

type deviceNode struct {
	path  string
	major uint32
	minor uint32
}

func (d deviceNode) devCharName() string {
	return fmt.Sprintf("%d:%d", d.major, d.minor)
}
