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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type command struct {
	logger logger.Interface
}

type options struct {
	folders       cli.StringSlice
	containerSpec string
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
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "folder",
			Usage:       "Specify a folder to add to /etc/ld.so.conf before updating the ld cache",
			Destination: &cfg.folders,
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) run(c *cli.Context, cfg *options) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRoot, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	ldconfigPath := m.resolveLDConfigPath("/sbin/ldconfig")
	args := []string{filepath.Base(ldconfigPath)}
	if containerRoot != "" {
		args = append(args, "-r", containerRoot)
	}

	if !root(containerRoot).hasPath("/etc/ld.so.cache") {
		m.logger.Debugf("No ld.so.cache found, skipping update")
		args = append(args, "-N")
	}

	folders := cfg.folders.Value()
	if root(containerRoot).hasPath("/etc/ld.so.conf.d") {
		err = m.createConfig(containerRoot, folders)
		if err != nil {
			return fmt.Errorf("failed to update ld.so.conf: %v", err)
		}
	} else {
		args = append(args, folders...)
	}

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	return syscall.Exec(ldconfigPath, args, nil)
}

type root string

func (r root) hasPath(path string) bool {
	_, err := os.Stat(filepath.Join(string(r), path))
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// resolveLDConfigPath determines the LDConfig path to use for the system.
// On systems such as Ubuntu where `/sbin/ldconfig` is a wrapper around
// /sbin/ldconfig.real, the latter is returned.
func (m command) resolveLDConfigPath(path string) string {
	return strings.TrimPrefix(config.NormalizeLDConfigPath("@"+path), "@")
}

// createConfig creates (or updates) /etc/ld.so.conf.d/nvcr-<RANDOM_STRING>.conf in the container
// to include the required paths.
func (m command) createConfig(root string, folders []string) error {
	if len(folders) == 0 {
		m.logger.Debugf("No folders to add to /etc/ld.so.conf")
		return nil
	}

	if err := os.MkdirAll(filepath.Join(root, "/etc/ld.so.conf.d"), 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %v", err)
	}

	configFile, err := os.CreateTemp(filepath.Join(root, "/etc/ld.so.conf.d"), "nvcr-*.conf")
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer configFile.Close()

	m.logger.Debugf("Adding folders %v to %v", folders, configFile.Name())

	configured := make(map[string]bool)
	for _, folder := range folders {
		if configured[folder] {
			continue
		}
		_, err = configFile.WriteString(fmt.Sprintf("%s\n", folder))
		if err != nil {
			return fmt.Errorf("failed to update ld.so.conf.d: %v", err)
		}
		configured[folder] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := os.Chmod(configFile.Name(), 0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %v", err)
	}

	return nil
}
