/*
 * Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dgxa100

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/gpus"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/server"
)

// Server is a type alias for server.Server maintained for backward compatibility.
//
// Deprecated: This type alias is maintained for backward compatibility only.
// The type may be removed in a future version.
type Server = server.Server

// Device is a type alias for server.Device maintained for backward compatibility.
//
// Deprecated: This type alias is maintained for backward compatibility only.
// The type may be removed in a future version.
type Device = server.Device

// GpuInstance is a type alias for server.GpuInstance maintained for backward compatibility.
//
// Deprecated: This type alias is maintained for backward compatibility only.
// The type may be removed in a future version.
type GpuInstance = server.GpuInstance

// ComputeInstance is a type alias for server.ComputeInstance maintained for backward compatibility.
//
// Deprecated: This type alias is maintained for backward compatibility only.
// The type may be removed in a future version.
type ComputeInstance = server.ComputeInstance

// CudaComputeCapability is a type alias for server.CudaComputeCapability maintained for backward compatibility.
//
// Deprecated: This type alias is maintained for backward compatibility only.
// The type may be removed in a future version.
type CudaComputeCapability = server.CudaComputeCapability

func New() *Server {
	return NewWithGPUs(gpus.Multiple(8, gpus.A100_SXM4_40GB)...)
}

func NewWithGPUs(gpus ...gpus.Config) *Server {
	s, _ := server.New(
		server.WithGPUs(gpus...),
		server.WithDriverVersion("550.54.15"),
		server.WithNVMLVersion("12.550.54.15"),
		server.WithCUDADriverVersion(12040),
	)
	return s
}

// Legacy globals for backward compatibility - expose the internal data
var (
	MIGProfiles = struct {
		GpuInstanceProfiles     map[int]nvml.GpuInstanceProfileInfo
		ComputeInstanceProfiles map[int]map[int]nvml.ComputeInstanceProfileInfo
	}{
		GpuInstanceProfiles:     gpus.A100_SXM4_40GB.MIGProfiles.GpuInstanceProfiles,
		ComputeInstanceProfiles: gpus.A100_SXM4_40GB.MIGProfiles.ComputeInstanceProfiles,
	}

	MIGPlacements = struct {
		GpuInstancePossiblePlacements     map[int][]nvml.GpuInstancePlacement
		ComputeInstancePossiblePlacements map[int]map[int][]nvml.ComputeInstancePlacement
	}{
		GpuInstancePossiblePlacements:     gpus.A100_SXM4_40GB.MIGProfiles.GpuInstancePlacements,
		ComputeInstancePossiblePlacements: gpus.A100_SXM4_40GB.MIGProfiles.ComputeInstancePlacements,
	}
)
