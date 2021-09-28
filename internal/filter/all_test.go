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
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	True := &SelectorMock{
		SelectedFunc: func(discover.Device) bool {
			return true
		},
	}
	False := &SelectorMock{
		SelectedFunc: func(discover.Device) bool {
			return false
		},
	}

	d := discover.Device{}

	// Ensure that the mocks are set up correctly:
	require.True(t, True.Selected(d))
	require.False(t, False.Selected(d))

	emtpy := All()
	require.True(t, emtpy.Selected(d))

	s00 := All(False, False)
	require.False(t, s00.Selected(d))

	s01 := All(False, True)
	require.False(t, s01.Selected(d))

	s10 := All(True, False)
	require.False(t, s10.Selected(d))

	s11 := All(True, True)
	require.True(t, s11.Selected(d))
}

type discoverMock struct {
	discover.None
	devices      []discover.Device
	devicesError error
	mounts       []discover.Mount
	mountsError  error
}

var _ discover.Discover = (*discoverMock)(nil)

func (m discoverMock) Devices() ([]discover.Device, error) {
	return m.devices, m.devicesError
}

func (m discoverMock) Mounts() ([]discover.Mount, error) {
	return m.mounts, m.mountsError
}
