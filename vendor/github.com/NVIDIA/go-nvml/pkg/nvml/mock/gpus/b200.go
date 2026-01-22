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

// B200 GPU Variants
var (
	B200_SXM5_180GB = Config{
		Name:         "NVIDIA B200 180GB HBM3e",
		Architecture: nvml.DEVICE_ARCH_BLACKWELL,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     184320, // 180GB
		CudaMajor:    10,
		CudaMinor:    0,
		PciDeviceId:  0x2B0010DE,
		MIGProfiles:  b200_180gb_MIGProfiles,
	}
)

var (
	b200_180gb_MIGProfiles = MIGProfileConfig{
		GpuInstanceProfiles:       b200_180gb_GpuInstanceProfiles,
		ComputeInstanceProfiles:   b200_ComputeInstanceProfiles,
		GpuInstancePlacements:     b200_GpuInstancePlacements,
		ComputeInstancePlacements: b200_ComputeInstancePlacements,
	}
)

var (
	b200_180gb_GpuInstanceProfiles = map[int]nvml.GpuInstanceProfileInfo{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE,
			IsP2pSupported:      1,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 18,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        23552, // 23GB (MIG 1g.23gb)
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1,
			IsP2pSupported:      1,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 18,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        1,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        23552, // 23GB (MIG 1g.23gb+me)
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2,
			IsP2pSupported:      1,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 18,
			CopyEngineCount:     1,
			DecoderCount:        1,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        46080, // 45GB (MIG 1g.45gb)
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_2_SLICE,
			IsP2pSupported:      1,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 36,
			CopyEngineCount:     2,
			DecoderCount:        2,
			EncoderCount:        1,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        46080, // 45GB (MIG 2g.45gb)
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_3_SLICE,
			IsP2pSupported:      1,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 54,
			CopyEngineCount:     3,
			DecoderCount:        3,
			EncoderCount:        2,
			JpegCount:           2,
			OfaCount:            2,
			MemorySizeMB:        92160, // 90GB (MIG 3g.90gb)
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_4_SLICE,
			IsP2pSupported:      1,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 72,
			CopyEngineCount:     4,
			DecoderCount:        4,
			EncoderCount:        2,
			JpegCount:           2,
			OfaCount:            2,
			MemorySizeMB:        92160, // 90GB (MIG 4g.90gb)
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_7_SLICE,
			IsP2pSupported:      1,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 126,
			CopyEngineCount:     7,
			DecoderCount:        7,
			EncoderCount:        4,
			JpegCount:           4,
			OfaCount:            4,
			MemorySizeMB:        184320, // 180GB (MIG 7g.180gb)
		},
	}
)

var b200_ComputeInstanceProfiles = map[int]map[int]nvml.ComputeInstanceProfileInfo{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 18,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 18,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       1,
			MultiprocessorCount: 18,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       2,
			MultiprocessorCount: 18,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 36,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       3,
			MultiprocessorCount: 18,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       1,
			MultiprocessorCount: 36,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
			SliceCount:          3,
			InstanceCount:       1,
			MultiprocessorCount: 54,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       4,
			MultiprocessorCount: 18,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       2,
			MultiprocessorCount: 36,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 72,
		},
	},
	nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
		nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 18,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 36,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 54,
		},
		nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE: {
			Id:                  nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 126,
		},
	},
}

var b200_GpuInstancePlacements = map[int][]nvml.GpuInstancePlacement{
	nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
		{Start: 0, Size: 1},
		{Start: 1, Size: 1},
		{Start: 2, Size: 1},
		{Start: 3, Size: 1},
		{Start: 4, Size: 1},
		{Start: 5, Size: 1},
		{Start: 6, Size: 1},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
		{Start: 0, Size: 1},
		{Start: 1, Size: 1},
		{Start: 2, Size: 1},
		{Start: 3, Size: 1},
		{Start: 4, Size: 1},
		{Start: 5, Size: 1},
		{Start: 6, Size: 1},
	},
	nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
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
		{Start: 0, Size: 7},
	},
}

var b200_ComputeInstancePlacements = map[int]map[int][]nvml.ComputeInstancePlacement{
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
			{Start: 0, Size: 7},
		},
	},
}
