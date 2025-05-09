/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package disabledevicenodemodification

import (
	"bufio"
	"bytes"
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

// NewCommand constructs an disable-device-node-modification subcommand with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

func (m command) build() *cli.Command {
	cfg := options{}

	c := cli.Command{
		Name:  "disable-device-node-modification",
		Usage: "Ensure that the /proc/driver/nvidia/params file present in the container does not allow device node modifications.",
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
	// TODO: Do we need to prefix the driver root?
	hostNvidiaParamsFile, err := os.Open(nvidiaDriverParamsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load params file: %w", err)
	}
	defer hostNvidiaParamsFile.Close()

	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRoot, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %w", err)
	}

	return m.updateNvidiaParamsFromReader(hostNvidiaParamsFile, containerRoot)
}

func (m command) updateNvidiaParamsFromReader(r io.Reader, containerRoot string) error {
	modifiedContents, err := m.getModifiedParamsFileContentsFromReader(r)
	if err != nil {
		return fmt.Errorf("failed to generate modified contents: %w", err)
	}
	if len(modifiedContents) == 0 {
		m.logger.Debugf("No modification required")
		return nil
	}
	return createParamsFileInContainer(containerRoot, modifiedContents)
}

// getModifiedParamsFileContentsFromReader returns the contents of a modified params file from the specified reader.
func (m command) getModifiedParamsFileContentsFromReader(r io.Reader) ([]byte, error) {
	var modified bytes.Buffer
	scanner := bufio.NewScanner(r)

	var requiresModification bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ModifyDeviceFiles: ") {
			if line == "ModifyDeviceFiles: 0" {
				m.logger.Debugf("Device node modification is already disabled")
				return nil, nil
			}
			if line == "ModifyDeviceFiles: 1" {
				line = "ModifyDeviceFiles: 0"
				requiresModification = true
			}
		}
		if _, err := modified.WriteString(line + "\n"); err != nil {
			return nil, fmt.Errorf("failed to create output buffer: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read params file: %w", err)
	}

	if !requiresModification {
		return nil, nil
	}

	return modified.Bytes(), nil
}

func createParamsFileInContainer(containerRoot string, contents []byte) error {
	if len(contents) == 0 {
		return nil
	}

	tempParamsFileName, err := createFileInTempfs("nvct-params", contents, 0o444)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	if err := bindMountReadonly(tempParamsFileName, filepath.Join(containerRoot, nvidiaDriverParamsPath)); err != nil {
		return fmt.Errorf("failed to create temporary params file mount: %w", err)
	}

	return nil
}

// createFileInTempfs creates a file with the specified name, contents, and mode in a tmpfs.
// A tmpfs is created at /tmp/nvct-emtpy-dir* with a size sufficient for the specified contents.
func createFileInTempfs(name string, contents []byte, mode os.FileMode) (string, error) {
	tmpRoot, err := os.MkdirTemp("", "nvct-empty-dir*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary folder: %w", err)
	}
	if err := createTmpFs(tmpRoot, len(contents)); err != nil {
		return "", fmt.Errorf("failed to create tmpfs mount for params file: %w", err)
	}

	filename := filepath.Join(tmpRoot, name)
	fileInTempfs, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary params file: %w", err)
	}
	defer fileInTempfs.Close()

	if _, err := fileInTempfs.Write(contents); err != nil {
		return "", fmt.Errorf("failed to write temporary params file: %w", err)
	}

	if err := fileInTempfs.Chmod(mode); err != nil {
		return "", fmt.Errorf("failed to set permissions on temporary params file: %w", err)
	}
	return filename, nil
}
