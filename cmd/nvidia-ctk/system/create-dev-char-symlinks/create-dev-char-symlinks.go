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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvdevices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvmodules"
)

const (
	defaultDevCharPath = "/dev/char"
)

type command struct {
	logger logger.Interface
}

type config struct {
	devCharPath       string
	driverRoot        string
	dryRun            bool
	watch             bool
	createAll         bool
	createDeviceNodes bool
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
	cfg := config{}

	// Create the 'create-dev-char-symlinks' command
	c := cli.Command{
		Name:  "create-dev-char-symlinks",
		Usage: "A utility to create symlinks to possible /dev/nv* devices in /dev/char",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
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
			Name:        "watch",
			Usage:       "If set, the command will watch for changes to the driver root and recreate the symlinks when changes are detected.",
			Value:       false,
			Destination: &cfg.watch,
			EnvVars:     []string{"WATCH"},
		},
		&cli.BoolFlag{
			Name:        "create-all",
			Usage:       "Create all possible /dev/char symlinks instead of limiting these to existing device nodes.",
			Destination: &cfg.createAll,
			EnvVars:     []string{"CREATE_ALL"},
		},
		&cli.BoolFlag{
			Name:        "load-kernel-modules",
			Usage:       "Load the NVIDIA kernel modules before creating symlinks. This is only applicable when --create-all is set.",
			Destination: &cfg.loadKernelModules,
			EnvVars:     []string{"LOAD_KERNEL_MODULES"},
		},
		&cli.BoolFlag{
			Name:        "create-device-nodes",
			Usage:       "Create the NVIDIA control device nodes in the driver root if they do not exist. This is only applicable when --create-all is set",
			Destination: &cfg.createDeviceNodes,
			EnvVars:     []string{"CREATE_DEVICE_NODES"},
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

func (m command) validateFlags(r *cli.Context, cfg *config) error {
	if cfg.createAll && cfg.watch {
		return fmt.Errorf("create-all and watch are mutually exclusive")
	}

	if cfg.loadKernelModules && !cfg.createAll {
		m.logger.Warning("load-kernel-modules is only applicable when create-all is set; ignoring")
		cfg.loadKernelModules = false
	}

	if cfg.createDeviceNodes && !cfg.createAll {
		m.logger.Warning("create-device-nodes is only applicable when create-all is set; ignoring")
		cfg.createDeviceNodes = false
	}

	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	var watcher *fsnotify.Watcher
	var sigs chan os.Signal

	if cfg.watch {
		watcher, err := newFSWatcher(filepath.Join(cfg.driverRoot, "dev"))
		if err != nil {
			return fmt.Errorf("failed to create FS watcher: %v", err)
		}
		defer watcher.Close()

		sigs = newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	}

	l, err := NewSymlinkCreator(
		WithLogger(m.logger),
		WithDevCharPath(cfg.devCharPath),
		WithDriverRoot(cfg.driverRoot),
		WithDryRun(cfg.dryRun),
		WithCreateAll(cfg.createAll),
		WithLoadKernelModules(cfg.loadKernelModules),
		WithCreateDeviceNodes(cfg.createDeviceNodes),
	)
	if err != nil {
		return fmt.Errorf("failed to create symlink creator: %v", err)
	}

create:
	err = l.CreateLinks()
	if err != nil {
		return fmt.Errorf("failed to create links: %v", err)
	}
	if !cfg.watch {
		return nil
	}
	for {
		select {

		case event := <-watcher.Events:
			deviceNode := filepath.Base(event.Name)
			if !strings.HasPrefix(deviceNode, "nvidia") {
				continue
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				m.logger.Infof("%s created, restarting.", event.Name)
				goto create
			}
			if event.Op&fsnotify.Create == fsnotify.Remove {
				m.logger.Infof("%s removed. Ignoring", event.Name)

			}

		// Watch for any other fs errors and log them.
		case err := <-watcher.Errors:
			m.logger.Errorf("inotify: %s", err)

		// React to signals
		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				m.logger.Infof("Received SIGHUP, recreating symlinks.")
				goto create
			default:
				m.logger.Infof("Received signal %q, shutting down.", s)
				return nil
			}
		}
	}
}

type linkCreator struct {
	logger            logger.Interface
	lister            nodeLister
	driverRoot        string
	devRoot           string
	devCharPath       string
	dryRun            bool
	createAll         bool
	createDeviceNodes bool
	loadKernelModules bool
}

// Creator is an interface for creating symlinks to /dev/nv* devices in /dev/char.
type Creator interface {
	CreateLinks() error
}

// Option is a functional option for configuring the linkCreator.
type Option func(*linkCreator)

// NewSymlinkCreator creates a new linkCreator.
func NewSymlinkCreator(opts ...Option) (Creator, error) {
	c := linkCreator{}
	for _, opt := range opts {
		opt(&c)
	}
	if c.logger == nil {
		c.logger = logger.New()
	}
	if c.driverRoot == "" {
		c.driverRoot = "/"
	}
	if c.devRoot == "" {
		c.devRoot = "/"
	}
	if c.devCharPath == "" {
		c.devCharPath = defaultDevCharPath
	}

	if err := c.setup(); err != nil {
		return nil, err
	}

	if c.createAll {
		lister, err := newAllPossible(c.logger, c.devRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to create all possible device lister: %v", err)
		}
		c.lister = lister
	} else {
		c.lister = existing{c.logger, c.devRoot}
	}
	return c, nil
}

func (m linkCreator) setup() error {
	if !m.loadKernelModules && !m.createDeviceNodes {
		return nil
	}

	if m.loadKernelModules {
		modules := nvmodules.New(
			nvmodules.WithLogger(m.logger),
			nvmodules.WithDryRun(m.dryRun),
			nvmodules.WithRoot(m.driverRoot),
		)
		if err := modules.LoadAll(); err != nil {
			return fmt.Errorf("failed to load NVIDIA kernel modules: %v", err)
		}
	}

	if m.createDeviceNodes {
		devices, err := nvdevices.New(
			nvdevices.WithLogger(m.logger),
			nvdevices.WithDryRun(m.dryRun),
			nvdevices.WithDevRoot(m.devRoot),
		)
		if err != nil {
			return err
		}
		if err := devices.CreateNVIDIAControlDevices(); err != nil {
			return fmt.Errorf("failed to create NVIDIA device nodes: %v", err)
		}
	}
	return nil
}

// WithDriverRoot sets the driver root path.
// This is the path in which kernel modules must be loaded.
func WithDriverRoot(root string) Option {
	return func(c *linkCreator) {
		c.driverRoot = root
	}
}

// WithDevRoot sets the root path for the /dev directory.
func WithDevRoot(root string) Option {
	return func(c *linkCreator) {
		c.devRoot = root
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
func WithLogger(logger logger.Interface) Option {
	return func(c *linkCreator) {
		c.logger = logger
	}
}

// WithCreateAll sets the createAll flag for the linkCreator.
func WithCreateAll(createAll bool) Option {
	return func(lc *linkCreator) {
		lc.createAll = createAll
	}
}

// WithLoadKernelModules sets the loadKernelModules flag for the linkCreator.
func WithLoadKernelModules(loadKernelModules bool) Option {
	return func(lc *linkCreator) {
		lc.loadKernelModules = loadKernelModules
	}
}

// WithCreateDeviceNodes sets the createDeviceNodes flag for the linkCreator.
func WithCreateDeviceNodes(createDeviceNodes bool) Option {
	return func(lc *linkCreator) {
		lc.createDeviceNodes = createDeviceNodes
	}
}

// CreateLinks creates symlinks for all NVIDIA device nodes found in the driver root.
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
			m.logger.Warningf("Could not create symlink: %v", err)
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

func newFSWatcher(files ...string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		err = watcher.Add(f)
		if err != nil {
			watcher.Close()
			return nil, err
		}
	}

	return watcher, nil
}

func newOSWatcher(sigs ...os.Signal) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	return sigChan
}
