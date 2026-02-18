/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestNew(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	defer devices.SetAllForTest()()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testCases := []struct {
		description       string
		mode              string
		rootfs            string
		expectedInitError error

		expectedSpec      *specs.Spec
		expectedSpecError error
	}{
		{
			description: "nvswitch mode is supported",
			mode:        "nvswitch",
			rootfs:      "rootfs-with-nvswitch",
			expectedSpec: &specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/nvswitch",
				Devices: []specs.Device{
					{
						Name: "all",
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path:     "/dev/nvidia-nvswitch0",
									HostPath: "/dev/nvidia-nvswitch0",
								},
								{
									Path:     "/dev/nvidia-nvswitch1",
									HostPath: "/dev/nvidia-nvswitch1",
								},
								{
									Path:     "/dev/nvidia-nvswitchctl",
									HostPath: "/dev/nvidia-nvswitchctl",
								},
							},
						},
					},
				},
				ContainerEdits: specs.ContainerEdits{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=void",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			driverRoot := filepath.Join(lookupRoot, tc.rootfs)

			l, err := New(
				WithLogger(logger),
				WithMode(tc.mode),
				WithDriverRoot(driverRoot),
			)
			require.EqualValues(t, tc.expectedInitError, err)

			s, err := l.GetSpec()
			require.EqualValues(t, tc.expectedSpecError, err)

			require.EqualValues(t, tc.expectedSpec, stripRoot(driverRoot, s.Raw()))
		})
	}
}
