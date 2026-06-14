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
)

// MIGProfiles holds the profile information for GIs and CIs in this mock server.
// We should consider auto-generating this object in the future.
var MIGProfiles = struct {
	GpuInstanceProfiles     map[int]nvml.GpuInstanceProfileInfo
	ComputeInstanceProfiles map[int]map[int]nvml.ComputeInstanceProfileInfo
}{
	GpuInstanceProfiles: map[int]nvml.GpuInstanceProfileInfo{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			Id:                  nvml.GPU_INSTANCE_PROFILE_1_SLICE,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 14,
			CopyEngineCount:     1,
			DecoderCount:        0,
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
			MultiprocessorCount: 98,
			CopyEngineCount:     7,
			DecoderCount:        5,
			EncoderCount:        0,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        40192,
		},
	},
	ComputeInstanceProfiles: map[int]map[int]nvml.ComputeInstanceProfileInfo{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    0,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    0,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         2,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 2,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         2,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 2,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         1,
				MultiprocessorCount:   28,
				SharedCopyEngineCount: 2,
				SharedDecoderCount:    1,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         3,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 3,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         3,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 3,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         1,
				MultiprocessorCount:   28,
				SharedCopyEngineCount: 3,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
				SliceCount:            3,
				InstanceCount:         1,
				MultiprocessorCount:   42,
				SharedCopyEngineCount: 3,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         4,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         4,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         2,
				MultiprocessorCount:   28,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE,
				SliceCount:            4,
				InstanceCount:         1,
				MultiprocessorCount:   56,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         7,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1,
				SliceCount:            1,
				InstanceCount:         7,
				MultiprocessorCount:   14,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         3,
				MultiprocessorCount:   28,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE,
				SliceCount:            3,
				InstanceCount:         2,
				MultiprocessorCount:   42,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE,
				SliceCount:            4,
				InstanceCount:         1,
				MultiprocessorCount:   56,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE: {
				Id:                    nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE,
				SliceCount:            7,
				InstanceCount:         1,
				MultiprocessorCount:   98,
				SharedCopyEngineCount: 7,
				SharedDecoderCount:    5,
				SharedEncoderCount:    0,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
		},
	},
}

// MIGPlacements holds the placement information for GIs and CIs in this mock server.
// We should consider auto-generating this object in the future.
var MIGPlacements = struct {
	GpuInstancePossiblePlacements     map[int][]nvml.GpuInstancePlacement
	ComputeInstancePossiblePlacements map[int]map[int][]nvml.ComputeInstancePlacement
}{
	GpuInstancePossiblePlacements: map[int][]nvml.GpuInstancePlacement{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			{
				Start: 0,
				Size:  1,
			},
			{
				Start: 1,
				Size:  1,
			},
			{
				Start: 2,
				Size:  1,
			},
			{
				Start: 3,
				Size:  1,
			},
			{
				Start: 4,
				Size:  1,
			},
			{
				Start: 5,
				Size:  1,
			},
			{
				Start: 6,
				Size:  1,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
			{
				Start: 0,
				Size:  1,
			},
			{
				Start: 1,
				Size:  1,
			},
			{
				Start: 2,
				Size:  1,
			},
			{
				Start: 3,
				Size:  1,
			},
			{
				Start: 4,
				Size:  1,
			},
			{
				Start: 5,
				Size:  1,
			},
			{
				Start: 6,
				Size:  1,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			{
				Start: 0,
				Size:  2,
			},
			{
				Start: 2,
				Size:  2,
			},
			{
				Start: 4,
				Size:  2,
			},
			{
				Start: 6,
				Size:  2,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			{
				Start: 0,
				Size:  2,
			},
			{
				Start: 2,
				Size:  2,
			},
			{
				Start: 4,
				Size:  2,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			{
				Start: 0,
				Size:  4,
			},
			{
				Start: 4,
				Size:  4,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			{
				Start: 0,
				Size:  4,
			},
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			{
				Start: 0,
				Size:  8,
			},
		},
	},
	// TODO: Fill out ComputeInstancePossiblePlacements
	ComputeInstancePossiblePlacements: map[int]map[int][]nvml.ComputeInstancePlacement{
		nvml.GPU_INSTANCE_PROFILE_1_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV1: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
		},
		nvml.GPU_INSTANCE_PROFILE_1_SLICE_REV2: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
		},
		nvml.GPU_INSTANCE_PROFILE_2_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE:      {},
		},
		nvml.GPU_INSTANCE_PROFILE_3_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE:      {},
		},
		nvml.GPU_INSTANCE_PROFILE_4_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE:      {},
		},
		nvml.GPU_INSTANCE_PROFILE_7_SLICE: {
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE_REV1: {},
			nvml.COMPUTE_INSTANCE_PROFILE_2_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_3_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_4_SLICE:      {},
			nvml.COMPUTE_INSTANCE_PROFILE_7_SLICE:      {},
		},
	},
}
