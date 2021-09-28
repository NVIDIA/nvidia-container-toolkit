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

package discover

import (
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvml"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/proc"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

const (
	testMajor = 999
)

func TestNVMLServerConstructor(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	nvml := nvml.NewMockNVMLOnLunaServer()

	d, err := createNVMLServer(logger, nvml, "")
	require.NoError(t, err)

	instance := d.(*nvmlServer)
	require.Len(t, instance.discoverers, 1+3+1)

	// We need to override the nvidiaDevices member of the nvmlDiscovery
	// TODO: Use a mock instead, or allow for injection into a constructor
	instance.discoverers[0].(*nvmlDiscover).nvidiaDevices = &mockNvidiaDevices{}

	devices, err := d.Devices()

	require.NoError(t, err)
	require.NotEmpty(t, devices)

	_, err = d.Mounts()
	require.NoError(t, err)
}

type mockNvidiaDevices struct{}

var _ proc.NvidiaDevices = (*mockNvidiaDevices)(nil)

func (d mockNvidiaDevices) Get(name string) (proc.Device, bool) {
	return proc.Device{Name: name, Major: testMajor}, true
}

func (d mockNvidiaDevices) Exists(string) bool {
	return false
}
