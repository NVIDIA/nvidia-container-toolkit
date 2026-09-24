/**
# SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package cudamemorylimits

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/cgroup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

type command struct {
	logger logger.Interface
}

const (
	GPUMemoryRequestEnvName = "NVIDIA_GPU_MEMORY_REQUEST"
	GPUMemoryLimitEnvName   = "NVIDIA_GPU_MEMORY_LIMIT"

	mebiByteMultiplier = 1024 * 1024
	maxMebiBytes       = nvml.DEVICE_MEMORY_LIMIT_MAX / mebiByteMultiplier
)

type config struct {
	driverRoot    string
	gpuIds        []string
	containerSpec string
}

func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

func (m command) build() *cli.Command {
	cfg := config{}

	c := cli.Command{
		Name: "apply-cuda-memory-limits",
		Usage: "Set the soft and hard limits of CUDA memory usage on GPU device(s) in the container. " +
			"It introspects the OCI container spec and fetches the memory limits from the environment variables:\n" +
			"1) NVIDIA_GPU_MEMORY_REQUEST\n2) NVIDIA_GPU_MEMORY_LIMIT\nThe values must be valid integers and are in MebiBytes.",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(cmd, &cfg)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.run(cmd, &cfg)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "driver-root",
				Usage:       "Specify the driver root",
				Destination: &cfg.driverRoot,
			},
			&cli.StringSliceFlag{
				Name:        "gpu-id",
				Usage:       "Specify the UUID of the GPU",
				Destination: &cfg.gpuIds,
			},
			&cli.StringFlag{
				Name:        "container-spec",
				Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
				Destination: &cfg.containerSpec,
			},
		},
	}

	return &c
}

func (m command) validateFlags(_ *cli.Command, cfg *config) error {
	if len(cfg.gpuIds) == 0 {
		return fmt.Errorf("at least one gpu-id must be specified")
	}
	for _, id := range cfg.gpuIds {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("gpu-id must not be empty")
		}
	}

	return nil
}

func (m command) run(_ *cli.Command, cfg *config) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}
	specFilePath := oci.GetSpecFilePath(s.Bundle)
	fs := oci.NewFileSpec(specFilePath, false)
	_, err = fs.Load()
	if err != nil {
		m.logger.Warningf("skipping apply-cuda-memory-limits; failed to load OCI container spec: %v", err)
		return nil
	}

	memReqStr, hasRequest := fs.LookupEnv(GPUMemoryRequestEnvName)
	memLimitStr, hasLimit := fs.LookupEnv(GPUMemoryLimitEnvName)
	if !hasRequest && !hasLimit {
		return nil
	}

	if !cgroup.IsCgroupV2() {
		return fmt.Errorf("setting GPU memory limits is only supported in cgroup v2")
	}

	cgroupPath, err := cgroup.GetAbsolutePath(s.Pid)
	if err != nil {
		return fmt.Errorf("failed to resolve cgroup path: %w", err)
	}

	var memoryRequestBytes, memoryLimitBytes uint64

	if hasRequest {
		memoryRequestMiB, err := parseMebiBytes(memReqStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", GPUMemoryRequestEnvName, err)
		}
		memoryRequestBytes = memoryRequestMiB * mebiByteMultiplier
	}
	if hasLimit {
		memoryLimitMiB, err := parseMebiBytes(memLimitStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", GPUMemoryLimitEnvName, err)
		}
		memoryLimitBytes = memoryLimitMiB * mebiByteMultiplier
	}

	if !hasLimit {
		memoryLimitBytes = nvml.DEVICE_MEMORY_LIMIT_MAX
	}
	if !hasRequest {
		memoryRequestBytes = memoryLimitBytes
	}

	if memoryRequestBytes > memoryLimitBytes {
		return fmt.Errorf("memory request (%d MiB) exceeds memory limit (%d MiB)",
			memoryRequestBytes/mebiByteMultiplier, memoryLimitBytes/mebiByteMultiplier)
	}

	return m.runApplyCudaMemoryLimits(cgroupPath, memoryRequestBytes, memoryLimitBytes, cfg.driverRoot, cfg.gpuIds)
}

func (m command) runApplyCudaMemoryLimits(cgroupPath string, requestBytes uint64, limitBytes uint64, driverRoot string, gpuIDs []string) error {
	driverLibLocator := lookup.NewLibraryLocator(
		lookup.WithLogger(m.logger),
		lookup.WithRoot(driverRoot),
	)

	candidates, err := driverLibLocator.Locate("libnvidia-ml.so.1")
	if err != nil {
		return fmt.Errorf("failed to locate libnvidia-ml.so.1: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no libnvidia-ml.so.1 found")
	}

	m.logger.Infof("driver library found: %s", candidates[0])

	nvmllib := nvml.New(nvml.WithLibraryPath(candidates[0]))
	ret := nvmllib.Init()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to initialize nvml: %w", ret)
	}
	defer func() {
		_ = nvmllib.Shutdown()
	}()

	for _, gpuID := range gpuIDs {
		device, ret := nvmllib.DeviceGetHandleByUUID(gpuID)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("failed to get GPU device handle with uuid %s: %w", gpuID, ret)
		}
		if device == nil {
			return fmt.Errorf("empty GPU device handle: %s", gpuID)
		}
		ret = device.SetMemoryLimits_v1(cgroupPath, requestBytes, limitBytes)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("failed to set memory limits for gpu %q: %w", gpuID, ret)
		}
	}
	return nil
}

func parseMebiBytes(value string) (uint64, error) {
	mebiBytes, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if mebiBytes > maxMebiBytes {
		return 0, fmt.Errorf("%d MiB exceeds the maximum supported value of %d MiB", mebiBytes, maxMebiBytes)
	}
	return mebiBytes, nil
}
