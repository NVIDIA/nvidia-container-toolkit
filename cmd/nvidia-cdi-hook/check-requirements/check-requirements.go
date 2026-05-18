/**
# Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package checkrequirements

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/requirements"
)

type command struct {
	logger logger.Interface
}

type options struct {
	containerSpec string
	driverRoot    string
}

// NewCommand constructs a check-requirements command with the specified logger.
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

func (m command) build() *cli.Command {
	cfg := options{}

	return &cli.Command{
		Name:  "check-requirements",
		Usage: "Check NVIDIA_REQUIRE_* constraints from the container image",
		Action: func(_ context.Context, _ *cli.Command) error {
			return m.run(&cfg)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "driver-root",
				Usage:       "Specify the NVIDIA GPU driver root to use when detecting host properties",
				Destination: &cfg.driverRoot,
			},
			&cli.StringFlag{
				Name:        "container-spec",
				Hidden:      true,
				Category:    "testing-only",
				Usage:       "Specify the path to the OCI container state. If empty or '-' the state will be read from STDIN",
				Destination: &cfg.containerSpec,
			},
		},
	}
}

func (m command) run(cfg *options) error {
	cudaImage, err := loadCUDAImageFromState(cfg.containerSpec, m.logger)
	if err != nil {
		return fmt.Errorf("failed to load CUDA image from container state: %w", err)
	}

	driver := root.New(
		root.WithLogger(m.logger),
		root.WithDriverRoot(cfg.driverRoot),
	)
	if err := requirements.CheckImage(m.logger, cudaImage, driver); err != nil {
		return fmt.Errorf("requirements not met: %w", err)
	}
	return nil
}

func loadCUDAImageFromState(containerStatePath string, logger logger.Interface) (*image.CUDA, error) {
	state, err := oci.LoadContainerState(containerStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load container state: %w", err)
	}

	specFilePath := oci.GetSpecFilePath(state.Bundle)
	specFile, err := os.Open(specFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OCI spec file: %w", err)
	}
	defer specFile.Close()

	var spec specs.Spec
	if err := json.NewDecoder(specFile).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode OCI spec: %w", err)
	}

	cudaImage, err := image.NewCUDAImageFromSpec(
		&spec,
		image.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}
	return &cudaImage, nil
}
