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

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestIPCDiscover(t *testing.T) {

	ipcLookup := map[string]string{
		"/var/run/nvidia-persistenced/socket":  "/var/run/nvidia-persistenced/socket",
		"/var/run/nvidia-fabricmanager/socket": "fm-socket",
	}

	logger, _ := testlog.NewNullLogger()

	d := NewIPCMountsWithLogger(logger, "")

	// Override lookup for testing
	mockLocator := NewLocatorMockFromMap(ipcLookup)
	d.(*mounts).lookup = mockLocator

	mounts, err := d.Mounts()
	require.NoError(t, err)
	require.ElementsMatch(t, []Mount{
		{Path: "/var/run/nvidia-persistenced/socket", Labels: map[string]string{}},
		{Path: "fm-socket", Labels: map[string]string{}}}, mounts)

	require.Len(t, mockLocator.LocateCalls(), 3)

	devices, err := d.Devices()
	require.NoError(t, err)
	require.Empty(t, devices)

	hooks, err := d.Hooks()
	require.NoError(t, err)
	require.Empty(t, hooks)
}
