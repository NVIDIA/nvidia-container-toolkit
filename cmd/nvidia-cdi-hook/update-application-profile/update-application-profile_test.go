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

package updateapplicationprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectGPUMinors(t *testing.T) {
	testCases := map[string]struct {
		nodes    []deviceNode
		expected []int
	}{
		"single GPU": {
			nodes:    []deviceNode{{name: "nvidia0", isCharDevice: true, minor: 0}},
			expected: []int{0},
		},
		"multiple GPUs, minor != filename order": {
			nodes: []deviceNode{
				{name: "nvidia0", isCharDevice: true, minor: 1},
				{name: "nvidia1", isCharDevice: true, minor: 0},
			},
			expected: []int{1, 0},
		},
		"multi-digit minor": {
			nodes:    []deviceNode{{name: "nvidia10", isCharDevice: true, minor: 10}},
			expected: []int{10},
		},
		"control and modeset nodes are excluded": {
			nodes: []deviceNode{
				{name: "nvidia0", isCharDevice: true, minor: 0},
				{name: "nvidiactl", isCharDevice: true, minor: 255},
				{name: "nvidia-modeset", isCharDevice: true, minor: 254},
				{name: "nvidia-uvm", isCharDevice: true, minor: 511},
				{name: "nvidia-uvm-tools", isCharDevice: true, minor: 510},
			},
			expected: []int{0},
		},
		"caps and imex nodes are excluded": {
			nodes: []deviceNode{
				{name: "nvidia-caps", isCharDevice: false, minor: 0},
				{name: "nvidia-caps-imex-channels", isCharDevice: false, minor: 0},
				{name: "nvidia1", isCharDevice: true, minor: 1},
			},
			expected: []int{1},
		},
		"non-char-device with matching name is excluded": {
			nodes: []deviceNode{
				{name: "nvidia0", isCharDevice: false, minor: 0},
			},
			expected: nil,
		},
		"unrelated device nodes are excluded": {
			nodes: []deviceNode{
				{name: "null", isCharDevice: true, minor: 3},
				{name: "zero", isCharDevice: true, minor: 5},
			},
			expected: nil,
		},
		"no nodes": {
			nodes:    nil,
			expected: nil,
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			require.Equal(t, tc.expected, selectGPUMinors(tc.nodes))
		})
	}
}

func TestBuildApplicationProfileConfig(t *testing.T) {
	testCases := map[string]struct {
		mask     uint64
		expected string
	}{
		"single GPU, minor 0": {
			mask:     1 << 0,
			expected: `{"profiles":[{"name":"_container_","settings":["EGLVisibleDGPUDevices",0x1]}],"rules":[{"pattern":[],"profile":"_container_"}]}`,
		},
		"minor 8": {
			mask:     1 << 8,
			expected: `{"profiles":[{"name":"_container_","settings":["EGLVisibleDGPUDevices",0x100]}],"rules":[{"pattern":[],"profile":"_container_"}]}`,
		},
		"minors 0 and 15 combined": {
			mask:     (1 << 0) | (1 << 15),
			expected: `{"profiles":[{"name":"_container_","settings":["EGLVisibleDGPUDevices",0x8001]}],"rules":[{"pattern":[],"profile":"_container_"}]}`,
		},
		"no GPUs": {
			mask:     0,
			expected: `{"profiles":[{"name":"_container_","settings":["EGLVisibleDGPUDevices",0x0]}],"rules":[{"pattern":[],"profile":"_container_"}]}`,
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			require.EqualValues(t, tc.expected, string(buildApplicationProfileConfig(tc.mask)))
		})
	}
}
