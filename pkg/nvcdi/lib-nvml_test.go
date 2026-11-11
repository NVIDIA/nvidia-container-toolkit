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
	mockserver "github.com/NVIDIA/go-nvml/pkg/nvml/mock/server"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
)

func TestNvmllibGetDeviceSpecGeneratorsForIDs(t *testing.T) {
	testCases := []struct {
		name               string
		ids                []string
		setupMock          func(*mockserver.Server)
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
			setupMock: func(server *mockserver.Server) {
				for _, d := range server.Devices {
					// TODO: This is not implemented in the mock.
					(d.(*mockserver.Device)).IsMigDeviceHandleFunc = func() (bool, nvml.Return) {
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
			setupMock: func(server *mockserver.Server) {
				for _, d := range server.Devices {
					// TODO: This is not implemented in the mock.
					(d.(*mockserver.Device)).IsMigDeviceHandleFunc = func() (bool, nvml.Return) {
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
			setupMock: func(server *mockserver.Server) {
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

				server.Devices[0].(*mockserver.Device).GetMigDeviceHandleByIndexFunc = func(n int) (nvml.Device, nvml.Return) {
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

func TestNvmllibDeviceSpecGeneratorsWithoutNVIDIADevices(t *testing.T) {
	testCases := []struct {
		name               string
		ids                []string
		initResult         nvml.Return
		expectedDeviceName string
		expectedInitCalls  int
		expectedError      error
	}{
		{
			name:               "all devices when the driver is not loaded",
			ids:                []string{"all"},
			initResult:         nvml.ERROR_DRIVER_NOT_LOADED,
			expectedDeviceName: "all",
			expectedInitCalls:  1,
		},
		{
			name:               "no devices when the driver is not loaded",
			ids:                []string{"none"},
			initResult:         nvml.ERROR_DRIVER_NOT_LOADED,
			expectedDeviceName: "none",
		},
		{
			name:              "preserve other NVML initialization errors",
			ids:               []string{"all"},
			initResult:        nvml.ERROR_NO_PERMISSION,
			expectedInitCalls: 1,
			expectedError:     nvml.ERROR_NO_PERMISSION,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockNvml := dgxa100.New()
			mockNvml.InitFunc = func() nvml.Return {
				return tc.initResult
			}

			l := &nvmllib{
				platformlibs: platformlibs{
					nvmllib:   mockNvml,
					devicelib: device.New(mockNvml),
				},
			}

			generators, err := l.DeviceSpecGenerators(tc.ids...)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				require.Nil(t, generators)
			} else {
				require.NoError(t, err)
				deviceSpecs, err := generators.GetDeviceSpecs()
				require.NoError(t, err)
				require.Len(t, deviceSpecs, 1)
				require.Equal(t, tc.expectedDeviceName, deviceSpecs[0].Name)
				require.Equal(t, []string{"NVCT_EMPTY_DEVICE="}, deviceSpecs[0].ContainerEdits.Env)
			}

			require.Len(t, mockNvml.InitCalls(), tc.expectedInitCalls)
		})
	}
}

// TODO: These need to be implemented in go-nvlib
func mockOverrides(server *mockserver.Server) {
	for i, d := range server.Devices {
		// TODO: This is not implemented in the mock.
		(d.(*mockserver.Device)).GetMaxMigDeviceCountFunc = func() (int, nvml.Return) {
			return 0, nvml.SUCCESS
		}
		(d.(*mockserver.Device)).GetIndexFunc = func() (int, nvml.Return) {
			return i, nvml.SUCCESS
		}
		(d.(*mockserver.Device)).GetUUIDFunc = func() (string, nvml.Return) {
			return d.(*mockserver.Device).UUID, nvml.SUCCESS
		}
	}
}
