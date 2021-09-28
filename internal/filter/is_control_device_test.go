/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package filter

import (
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestControlDevice(t *testing.T) {
	control := ControlDevice()

	pcibusID := discover.PCIBusID("pcibusid")
	device := discover.Device{
		Index:    "index",
		UUID:     "uuid",
		PCIBusID: pcibusID,
	}
	require.False(t, control.Selected(device))

	require.False(t, control.Selected(
		discover.Device{UUID: "uuid", PCIBusID: pcibusID},
	))

	require.False(t, control.Selected(
		discover.Device{Index: "index", PCIBusID: pcibusID},
	))

	require.False(t, control.Selected(
		discover.Device{Index: "index", UUID: "uuid"},
	))

	require.False(t, control.Selected(discover.Device{}))

	require.True(t, control.Selected(discover.Device{UUID: "uuid"}))
}

func TestGetControlDevicesFromEnv(t *testing.T) {
	testCases := []struct {
		name        string
		env         map[string]string
		expectedIds []string
	}{
		{
			name:        "nil environment",
			env:         nil,
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "empty environment",
			env:         map[string]string{},
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "MIG monitor blank",
			env:         map[string]string{"NVIDIA_MIG_MONITOR_DEVICES": ""},
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "MIG config blank",
			env:         map[string]string{"NVIDIA_MIG_CONFIG_DEVICES": ""},
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "MIG monitor not all",
			env:         map[string]string{"NVIDIA_MIG_MONITOR_DEVICES": "not-all"},
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "MIG config not all",
			env:         map[string]string{"NVIDIA_MIG_CONFIG_DEVICES": "not-all"},
			expectedIds: []string{"CONTROL"},
		},
		{
			name:        "MIG monitor all",
			env:         map[string]string{"NVIDIA_MIG_MONITOR_DEVICES": "all"},
			expectedIds: []string{"CONTROL", "MONITOR"},
		},
		{
			name:        "MIG config all",
			env:         map[string]string{"NVIDIA_MIG_CONFIG_DEVICES": "all"},
			expectedIds: []string{"CONTROL", "CONFIG"},
		},
		{
			name:        "MIG config and monitor all",
			env:         map[string]string{"NVIDIA_MIG_CONFIG_DEVICES": "all", "NVIDIA_MIG_MONITOR_DEVICES": "all"},
			expectedIds: []string{"CONTROL", "CONFIG", "MONITOR"},
		},
	}

	for i, tc := range testCases {
		logger, _ := testlog.NewNullLogger()

		t.Run(fmt.Sprintf("%d: %s", i, tc.name), func(t *testing.T) {
			deviceIDs := getControlDeviceIDsFromEnvWithLogger(logger, &EnvLookupMock{
				LookupEnvFunc: func(s string) (string, bool) {
					value, exists := tc.env[s]
					return value, exists
				},
			})
			require.ElementsMatch(t, tc.expectedIds, deviceIDs)
		})
	}

}
