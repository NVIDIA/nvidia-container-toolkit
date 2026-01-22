/*
 * Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package gpus

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// A30 GPU Variants with different memory profiles and PCI device IDs
var (
	A30_PCIE_24GB = Config{
		Name:         "NVIDIA A30-PCIE-24GB",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     24576,
		CudaMajor:    8,
		CudaMinor:    0,
		PciDeviceId:  0x20B710DE,
		MIGProfiles:  a30_24gb_MIGProfiles,
	}
)

var a30_24gb_MIGProfiles = MIGProfileConfig{
	GpuInstanceProfiles:       a30_24gb_GpuInstanceProfiles,
	ComputeInstanceProfiles:   a30_ComputeInstanceProfiles,
	GpuInstancePlacements:     a30_GpuInstancePlacements,
	ComputeInstancePlacements: a30_ComputeInstancePlacements,
}

var a30_24gb_GpuInstanceProfiles = map[int]nvml.GpuInstanceProfileInfo{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE,
		IsP2pSupported:      0,
		SliceCount:          1,
		InstanceCount:       4,
		MultiprocessorCount: 14,
		CopyEngineCount:     1,
		DecoderCount:        0,
		EncoderCount:        0,
		JpegCount:           0,
		OfaCount:            0,
		MemorySizeMB:        5836,
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1,
		IsP2pSupported:      0,
		SliceCount:          1,
		InstanceCount:       1,
		MultiprocessorCount: 14,
		CopyEngineCount:     1,
		DecoderCount:        1,
		EncoderCount:        0,
		JpegCount:           1,
		OfaCount:            1,
		MemorySizeMB:        5836,
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		Id:                  nvml.GPU_INSTANCE_PROFILE_2_SLICE,
		IsP2pSupported:      0,
		SliceCount:          2,
		InstanceCount:       2,
		MultiprocessorCount: 28,
		CopyEngineCount:     2,
		DecoderCount:        2,
		EncoderCount:        0,
		JpegCount:           0,
		OfaCount:            0,
		MemorySizeMB:        11672,
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE_REV1: {
		Id:                  nvml.GPU_INSTANCE_PROFILE_2_SLICE_REV1,
		IsP2pSupported:      0,
		SliceCount:          2,
		InstanceCount:       1,
		MultiprocessorCount: 28,
		CopyEngineCount:     2,
		DecoderCount:        2,
		EncoderCount:        0,
		JpegCount:           1,
		OfaCount:            1,
		MemorySizeMB:        11672,
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		Id:                  nvml.GPU_INSTANCE_PROFILE_4_SLICE,
		IsP2pSupported:      0,
		SliceCount:          4,
		InstanceCount:       1,
		MultiprocessorCount: 56,
		CopyEngineCount:     4,
		DecoderCount:        4,
		EncoderCount:        0,
		JpegCount:           1,
		OfaCount:            1,
		MemorySizeMB:        23344,
	},
}

var a30_ComputeInstanceProfiles = map[int]map[int]nvml.ComputeInstanceProfileInfo{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 14,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 14,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       2,
			MultiprocessorCount: 14,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 28,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE_REV1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       2,
			MultiprocessorCount: 14,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 28,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 14,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       2,
			MultiprocessorCount: 28,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 56,
		},
	},
}

var a30_GpuInstancePlacements = map[int][]nvml.GpuInstancePlacement{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		{Start: 0, Size: 1},
		{Start: 1, Size: 1},
		{Start: 2, Size: 1},
		{Start: 3, Size: 1},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		{Start: 0, Size: 1},
		{Start: 1, Size: 1},
		{Start: 2, Size: 1},
		{Start: 3, Size: 1},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		{Start: 0, Size: 2},
		{Start: 2, Size: 2},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE_REV1: {
		{Start: 0, Size: 2},
		{Start: 2, Size: 2},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		{Start: 0, Size: 4},
	},
}

var a30_ComputeInstancePlacements = map[int]map[int][]nvml.ComputeInstancePlacement{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
		},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE_REV1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
		},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
			{Start: 2, Size: 1},
			{Start: 3, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
			{Start: 2, Size: 2},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
			{Start: 0, Size: 4},
		},
	},
}
