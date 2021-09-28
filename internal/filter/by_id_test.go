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

func TestDeviceByID(t *testing.T) {
	device := discover.Device{
		Index:    "index",
		UUID:     "uuid",
		PCIBusID: discover.PCIBusID("pcibusid"),
	}

	require.False(t, NewDeviceSelector().Selected(device))
	require.False(t, NewDeviceSelector("notindex", "notuuid", "notpcibusid").Selected(device))

	require.True(t, NewDeviceSelector("index").Selected(device))
	require.True(t, NewDeviceSelector("notindex", "index").Selected(device))

	require.True(t, NewDeviceSelector("uuid").Selected(device))
	require.True(t, NewDeviceSelector("notuuid", "uuid").Selected(device))

	require.True(t, NewDeviceSelector("pcibusid").Selected(device))
	require.True(t, NewDeviceSelector("notpcibusid", "pcibusid").Selected(device))

	require.True(t, NewDeviceSelector("index", "uuid", "pcibusid").Selected(device))

}
