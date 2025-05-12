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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	// ldsoconfdFilenamePattern specifies the pattern for the filename
	// in ld.so.conf.d that includes references to the specified directories.
	// The 00-nvcr prefix is chosen to ensure that these libraries have a
	// higher precedence than other libraries on the system, but lower than
	// the 00-cuda-compat that is included in some containers.
	ldsoconfdFilenamePattern = "00-nvcr-*.conf"
)

type command struct {
	logger logger.Interface
}

type options struct {
	folders       cli.StringSlice
	ldconfigPath  string
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

	ldconfigPath := m.resolveLDConfigPath(cfg.ldconfigPath)
	args := []string{
		filepath.Base(ldconfigPath),
		// Run ldconfig in the container root directory on the host.
		"-r", containerRootDir,
		// Explicitly specify using /etc/ld.so.conf since the host's ldconfig may
		// be configured to use a different config file by default.
		// Note that since we apply the `-r {{ .containerRootDir }}` argument, /etc/ld.so.conf is
		// in the container.
		"-f", "/etc/ld.so.conf",
	}

	containerRoot := containerRoot(containerRootDir)

	if containerRoot.hasPath("/etc/ld.so.cache") {
		args = append(args, "-C", "/etc/ld.so.cache")
	} else {
		m.logger.Debugf("No ld.so.cache found, skipping update")
		args = append(args, "-N")
	}

	folders := cfg.folders.Value()
	if containerRoot.hasPath("/etc/ld.so.conf.d") {
		err := m.createLdsoconfdFile(containerRoot, ldsoconfdFilenamePattern, folders...)
		if err != nil {
			return fmt.Errorf("failed to update ld.so.conf.d: %v", err)
		}
	} else {
		args = append(args, folders...)
	}

	return m.SafeExec(ldconfigPath, args, nil)
}

// resolveLDConfigPath determines the LDConfig path to use for the system.
// On systems such as Ubuntu where `/sbin/ldconfig` is a wrapper around
// /sbin/ldconfig.real, the latter is returned.
func (m command) resolveLDConfigPath(path string) string {
	return strings.TrimPrefix(config.NormalizeLDConfigPath("@"+path), "@")
}

// createLdsoconfdFile creates a file at /etc/ld.so.conf.d/ in the specified root.
// The file is created at /etc/ld.so.conf.d/{{ .pattern }} using `CreateTemp` and
// contains the specified directories on each line.
func (m command) createLdsoconfdFile(in containerRoot, pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		m.logger.Debugf("No directories to add to /etc/ld.so.conf")
		return nil
	}

	ldsoconfdDir, err := in.resolve("/etc/ld.so.conf.d")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	m.logger.Debugf("Adding directories %v to %v", dirs, configFile.Name())

	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err = fmt.Fprintf(configFile, "%s\n", dir)
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}
