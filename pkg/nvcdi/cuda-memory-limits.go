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

package nvcdi

import (
	"fmt"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type cudaMemoryLimits struct {
	logger      logger.Interface
	driverRoot  string
	uuid        string
	hookCreator discover.HookCreator
}

func (l *nvcdilib) newCudaMemoryLimits(d device.Device) (discover.Discover, error) {
	uuid, nvmlRet := d.GetUUID()
	if nvmlRet != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device UUID: %w", nvmlRet)
	}

	cMemLimits := &cudaMemoryLimits{
		logger:      l.logger,
		driverRoot:  l.driver.Root,
		uuid:        uuid,
		hookCreator: l.hookCreator,
	}

	return cMemLimits, nil
}

// Devices are empty for this discoverer
func (c *cudaMemoryLimits) Devices() ([]discover.Device, error) {
	return nil, nil
}

// EnvVars are empty for this discoverer
func (c *cudaMemoryLimits) EnvVars() ([]discover.EnvVar, error) {
	return nil, nil
}

// Hooks returns a set of hooks that assigns a CUDA memory limit to the cgroup of the GPU workload container
func (c *cudaMemoryLimits) Hooks() ([]discover.Hook, error) {
	return c.hookCreator.Create("apply-cuda-memory-limits", c.driverRoot, c.uuid).Hooks()
}

// Mounts are empty for this discoverer
func (c *cudaMemoryLimits) Mounts() ([]discover.Mount, error) {
	return nil, nil
}
