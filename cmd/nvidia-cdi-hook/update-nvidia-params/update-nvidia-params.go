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

package nvidiaparams

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	nvidiaDriverParamsPath = "/proc/driver/nvidia/params"
)

type command struct {
	logger logger.Interface
}

type options struct {
	containerSpec string
}

// NewCommand constructs an update-nvidia-params command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the update-nvidia-params command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'update-nvidia-params' command
	c := cli.Command{
		Name:  "update-nvidia-params",
		Usage: "Update ldcache in a container by running ldconfig",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "container-spec",
			Hidden:      true,
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, cfg *options) error {
	return nil
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

	return m.updateNvidiaParams(containerRoot)
}

func (m command) updateNvidiaParams(containerRoot string) error {
	// TODO: Do we need to prefix the driver root?
	currentParamsFile, err := os.Open(nvidiaDriverParamsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load params file: %w", err)
	}
	defer currentParamsFile.Close()

	return m.updateNvidiaParamsFromReader(currentParamsFile, containerRoot)
}

func (m command) updateNvidiaParamsFromReader(r io.Reader, containerRoot string) error {
	var newLines []string
	scanner := bufio.NewScanner(r)
	var requiresModification bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ModifyDeviceFiles: ") {
			if line == "ModifyDeviceFiles: 0" {
				m.logger.Debugf("Device node modification is already disabled; exiting")
				return nil
			}
			if line == "ModifyDeviceFiles: 1" {
				line = "ModifyDeviceFiles: 0"
				requiresModification = true
			}
		}
		newLines = append(newLines, line)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read params file: %w", err)
	}

	if !requiresModification {
		return nil
	}

	containerParamsFile, err := os.CreateTemp("", "nvct-params-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary params file: %w", err)
	}
	defer containerParamsFile.Close()

	if _, err := containerParamsFile.WriteString(strings.Join(newLines, "\n")); err != nil {
		return fmt.Errorf("failed to write temporary params file: %w", err)
	}

	if err := containerParamsFile.Chmod(0o644); err != nil {
		return fmt.Errorf("failed to set permissions on temporary params file: %w", err)
	}

	if err := bindMountReadonly(containerParamsFile.Name(), filepath.Join(containerRoot, nvidiaDriverParamsPath)); err != nil {
		return fmt.Errorf("failed to create temporary parms file mount: %w", err)
	}

	return nil
}
