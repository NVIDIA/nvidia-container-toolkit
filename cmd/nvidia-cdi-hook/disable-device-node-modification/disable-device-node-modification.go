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
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	nvidiaDriverParamsPath = "/proc/driver/nvidia/params"
)

type options struct {
	containerSpec string
}

// NewCommand constructs an disable-device-node-modification subcommand with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	cfg := options{}

	c := cli.Command{
		Name:  "disable-device-node-modification",
		Usage: "Ensure that the /proc/driver/nvidia/params file present in the container does not allow device node modifications.",
		Before: func(c *cli.Context) error {
			return validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return run(c, &cfg)
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

func validateFlags(c *cli.Context, cfg *options) error {
	return nil
}

func run(_ *cli.Context, cfg *options) error {
	modifiedParamsFileContents, err := getModifiedNVIDIAParamsContents()
	if err != nil {
		return fmt.Errorf("failed to get modified params file contents: %w", err)
	}
	if len(modifiedParamsFileContents) == 0 {
		return nil
	}

	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRootDirPath, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %w", err)
	}

	return createParamsFileInContainer(containerRootDirPath, modifiedParamsFileContents)
}

func getModifiedNVIDIAParamsContents() ([]byte, error) {
	hostNvidiaParamsFile, err := os.Open(nvidiaDriverParamsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load params file: %w", err)
	}
	defer hostNvidiaParamsFile.Close()

	modifiedContents, err := getModifiedParamsFileContentsFromReader(hostNvidiaParamsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get modfied params file contents: %w", err)
	}

	return modifiedContents, nil
}

// getModifiedParamsFileContentsFromReader returns the contents of a modified params file from the specified reader.
func getModifiedParamsFileContentsFromReader(r io.Reader) ([]byte, error) {
	var modified bytes.Buffer
	scanner := bufio.NewScanner(r)

	var requiresModification bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ModifyDeviceFiles: ") {
			if line == "ModifyDeviceFiles: 0" {
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
