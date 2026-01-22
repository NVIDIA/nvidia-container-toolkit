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

// A100 GPU Variants with different memory profiles and PCI device IDs
var (
	A100_PCIE_40GB = Config{
		Name:         "NVIDIA A100-PCIE-40GB",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     40960,
		CudaMajor:    8,
		CudaMinor:    0,
		PciDeviceId:  0x20F110DE,
		MIGProfiles:  a100_40gb_MIGProfiles,
	}
	A100_PCIE_80GB = Config{
		Name:         "NVIDIA A100-PCIE-80GB",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     81920,
		CudaMajor:    8,
		CudaMinor:    0,
		PciDeviceId:  0x20B510DE,
		MIGProfiles:  a100_80gb_MIGProfiles,
	}
	A100_SXM4_40GB = Config{
		Name:         "Mock NVIDIA A100-SXM4-40GB",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     40960,
		CudaMajor:    8,
		CudaMinor:    0,
		PciDeviceId:  0x20B010DE,
		MIGProfiles:  a100_40gb_MIGProfiles,
	}
	A100_SXM4_80GB = Config{
		Name:         "NVIDIA A100-SXM4-80GB",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     81920,
		CudaMajor:    8,
		CudaMinor:    0,
		PciDeviceId:  0x20B210DE,
		MIGProfiles:  a100_80gb_MIGProfiles,
	}
)

var (
	a100_40gb_MIGProfiles = MIGProfileConfig{
		GpuInstanceProfiles:       a100_40gb_GpuInstanceProfiles,
		ComputeInstanceProfiles:   a100_ComputeInstanceProfiles,
		GpuInstancePlacements:     a100_GpuInstancePlacements,
		ComputeInstancePlacements: a100_ComputeInstancePlacements,
	}
	a100_80gb_MIGProfiles = MIGProfileConfig{
		GpuInstanceProfiles:       a100_80gb_GpuInstanceProfiles,
		ComputeInstanceProfiles:   a100_ComputeInstanceProfiles,
		GpuInstancePlacements:     a100_GpuInstancePlacements,
		ComputeInstancePlacements: a100_ComputeInstancePlacements,
	}
)

var (
	a100_40gb_GpuInstanceProfiles = map[int]nvml.GpuInstanceProfileInfo{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 14,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        4864,
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
			MemorySizeMB:        4864,
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 14,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        9856,
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_2_SLICE,
			IsP2pSupported:      0,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 28,
			CopyEngineCount:     2,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        9856,
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_3_SLICE,
			IsP2pSupported:      0,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 42,
			CopyEngineCount:     3,
			DecoderCount:        2,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        19968,
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_4_SLICE,
			IsP2pSupported:      0,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 56,
			CopyEngineCount:     4,
			DecoderCount:        2,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        19968,
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_7_SLICE,
			IsP2pSupported:      0,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 96,
			CopyEngineCount:     7,
			DecoderCount:        5,
			EncoderCount:        0,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        40192,
		},
	}
	a100_80gb_GpuInstanceProfiles = map[int]nvml.GpuInstanceProfileInfo{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 14,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        9856,
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
			MemorySizeMB:        9856,
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 14,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        19968,
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_2_SLICE,
			IsP2pSupported:      0,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 28,
			CopyEngineCount:     2,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        19968,
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_3_SLICE,
			IsP2pSupported:      0,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 42,
			CopyEngineCount:     3,
			DecoderCount:        2,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        40192,
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_4_SLICE,
			IsP2pSupported:      0,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 56,
			CopyEngineCount:     4,
			DecoderCount:        2,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        40192,
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_7_SLICE,
			IsP2pSupported:      0,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 98,
			CopyEngineCount:     7,
			DecoderCount:        5,
			EncoderCount:        0,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        80384,
		},
	}
)

var a100_ComputeInstanceProfiles = map[int]map[int]nvml.ComputeInstanceProfileInfo{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 16,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       2,
			MultiprocessorCount: 16,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 32,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       3,
			MultiprocessorCount: 16,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 32,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
			SliceCount:          3,
			InstanceCount:       1,
			MultiprocessorCount: 48,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 16,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       2,
			MultiprocessorCount: 32,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 64,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 16,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 32,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 48,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 112,
		},
	},
}

var a100_GpuInstancePlacements = map[int][]nvml.GpuInstancePlacement{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		{Start: 0, Size: 1},
		{Start: 1, Size: 1},
		{Start: 2, Size: 1},
		{Start: 3, Size: 1},
		{Start: 4, Size: 1},
		{Start: 5, Size: 1},
		{Start: 6, Size: 1},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		{Start: 0, Size: 2},
		{Start: 2, Size: 2},
		{Start: 4, Size: 2},
	},
	nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
		{Start: 0, Size: 3},
		{Start: 4, Size: 3},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		{Start: 0, Size: 4},
	},
	nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
		{Start: 0, Size: 8}, // Test expects Size 8
	},
}

var a100_ComputeInstancePlacements = map[int]map[int][]nvml.ComputeInstancePlacement{
	0: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
		},
	},
	1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
		},
	},
	2: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
			{Start: 2, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			{Start: 0, Size: 3},
		},
	},
	3: {
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
	4: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			{Start: 0, Size: 1},
			{Start: 1, Size: 1},
			{Start: 2, Size: 1},
			{Start: 3, Size: 1},
			{Start: 4, Size: 1},
			{Start: 5, Size: 1},
			{Start: 6, Size: 1},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			{Start: 0, Size: 2},
			{Start: 2, Size: 2},
			{Start: 4, Size: 2},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			{Start: 0, Size: 3},
			{Start: 4, Size: 3},
		},
		nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE: {
			{Start: 0, Size: 8}, // Test expects Size 8
		},
	},
}
