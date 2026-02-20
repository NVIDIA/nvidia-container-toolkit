/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	mocknvml "github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxa100"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
)

func TestNvmllibGetDeviceSpecGeneratorsForIDs(t *testing.T) {
	testCases := []struct {
		name               string
		ids                []string
		setupMock          func(*dgxa100.Server)
		expectedError      error
		expectedLength     int
		expectedGenerators DeviceSpecGenerators
	}{
		{
			name:           "all devices",
			ids:            []string{"all"},
			expectedError:  nil,
			expectedLength: 8,
		},
		{
			name: "single GPU index",
			ids:  []string{"0"},
			setupMock: func(server *dgxa100.Server) {
				for _, d := range server.Devices {
					// TODO: This is not implemented in the mock.
					(d.(*dgxa100.Device)).IsMigDeviceHandleFunc = func() (bool, nvml.Return) {
						return false, nvml.SUCCESS
					}
				}
			},
			expectedError:  nil,
			expectedLength: 1,
		},
		{
			name: "single UUID",
			ids:  []string{"GPU-12345678-1234-1234-1234-123456789abc"},
			setupMock: func(server *dgxa100.Server) {
				for _, d := range server.Devices {
					// TODO: This is not implemented in the mock.
					(d.(*dgxa100.Device)).IsMigDeviceHandleFunc = func() (bool, nvml.Return) {
						return false, nvml.SUCCESS
					}
				}
				server.DeviceGetHandleByUUIDFunc = func(s string) (nvml.Device, nvml.Return) {
					if s == "GPU-12345678-1234-1234-1234-123456789abc" {
						return server.Devices[3], nvml.SUCCESS
					}
					return server.Devices[0], nvml.SUCCESS
				}
			},
			expectedError:  nil,
			expectedLength: 1,
		},
		{
			name: "MIG device index",
			ids:  []string{"0:0"},
			setupMock: func(server *dgxa100.Server) {
				mig := &mocknvml.Device{
					IsMigDeviceHandleFunc: func() (bool, nvml.Return) {
						return true, nvml.SUCCESS
					},
					GetDeviceHandleFromMigDeviceHandleFunc: func() (nvml.Device, nvml.Return) {
						return server.Devices[0], nvml.SUCCESS
					},
					GetIndexFunc: func() (int, nvml.Return) {
						return 0, nvml.SUCCESS
					},
					GetUUIDFunc: func() (string, nvml.Return) {
						return "MIG-foo", nvml.SUCCESS
					},
				}

				server.Devices[0].(*dgxa100.Device).GetMigDeviceHandleByIndexFunc = func(n int) (nvml.Device, nvml.Return) {
					if n != 0 {
						return nil, nvml.ERROR_INVALID_ARGUMENT
					}

					return mig, nvml.SUCCESS
				}

				server.DeviceGetHandleByUUIDFunc = func(s string) (nvml.Device, nvml.Return) {
					if s == "MIG-foo" {
						return mig, nvml.SUCCESS
					}
					return nil, nvml.ERROR_INVALID_ARGUMENT
				}
			},
			expectedError:  nil,
			expectedLength: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a mock server, using the DGX A100 mock.
			mockNvml := dgxa100.New()
			mockOverrides(mockNvml)
			if tc.setupMock != nil {
				tc.setupMock(mockNvml)
			}

			mockDev := device.New(mockNvml)

			l := &nvmllib{
				platformlibs: platformlibs{
					nvmllib:   mockNvml,
					devicelib: mockDev,
				},
			}
			// Call the function under test
			generators, err := l.getDeviceSpecGeneratorsForIDs(tc.ids...)

			require.EqualValues(t, tc.expectedError, err)
			require.Len(t, generators, tc.expectedLength)
		})
	}
}

// TODO: These need to be implemented in go-nvlib
func mockOverrides(server *dgxa100.Server) {
	for i, d := range server.Devices {
		// TODO: This is not implemented in the mock.
		(d.(*dgxa100.Device)).GetMaxMigDeviceCountFunc = func() (int, nvml.Return) {
			return 0, nvml.SUCCESS
		}
		(d.(*dgxa100.Device)).GetIndexFunc = func() (int, nvml.Return) {
			return i, nvml.SUCCESS
		}
		(d.(*dgxa100.Device)).GetUUIDFunc = func() (string, nvml.Return) {
			return d.(*dgxa100.Device).UUID, nvml.SUCCESS
		}
	}
}
