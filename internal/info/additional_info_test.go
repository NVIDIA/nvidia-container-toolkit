/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package info

import (
	"testing"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
	"github.com/stretchr/testify/require"
)

func TestUsesNVGPUModule(t *testing.T) {
	testCases := []struct {
		description string
		nvmllib     nvml.Interface
		expected    bool
	}{
		{
			description: "init failure returns false",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.ERROR_LIBRARY_NOT_FOUND
				},
			},
			expected: false,
		},
		{
			description: "no devices returns false",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 0, nvml.SUCCESS
				},
			},
			expected: false,
		},
		{
			description: "DeviceGetCount error returns false",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 0, nvml.ERROR_UNKNOWN
				},
			},
			expected: false,
		},
		{
			description: "Failure to get device name returns false",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 1, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return "", nvml.ERROR_UNKNOWN
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: false,
		},
		{
			description: "nested panic returns false",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 1, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							panic("deep panic")
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: false,
		},
		{
			description: "Single device name with no nvgpu",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 1, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return "NVIDIA A100-SXM4-40GB", nvml.SUCCESS
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: false,
		},
		{
			description: "Single device name with nvgpu",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 1, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return "Orin (nvgpu)", nvml.SUCCESS
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: true,
		},
		{
			description: "Multiple device names with no nvgpu",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 2, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return "NVIDIA A100-SXM4-40GB", nvml.SUCCESS
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: false,
		},
		{
			description: "Multiple device names with nvgpu",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 2, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return "Orin (nvgpu)", nvml.SUCCESS
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: true,
		},
		{
			description: "Mixed device names",
			nvmllib: &nvml.InterfaceMock{
				InitFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				ShutdownFunc: func() nvml.Return {
					return nvml.SUCCESS
				},
				DeviceGetCountFunc: func() (int, nvml.Return) {
					return 2, nvml.SUCCESS
				},
				DeviceGetHandleByIndexFunc: func(index int) (nvml.Device, nvml.Return) {
					var deviceName string
					if index == 0 {
						deviceName = "NVIDIA A100-SXM4-40GB"
					} else {
						deviceName = "Orin (nvgpu)"
					}
					device := &nvml.DeviceMock{
						GetNameFunc: func() (string, nvml.Return) {
							return deviceName, nvml.SUCCESS
						},
					}
					return device, nvml.SUCCESS
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			sut := additionalInfo{
				nvmllib:   tc.nvmllib,
				devicelib: device.New(device.WithNvml(tc.nvmllib)),
			}

			flag, _ := sut.UsesNVGPUModule()
			require.Equal(t, tc.expected, flag)
		})
	}
}
