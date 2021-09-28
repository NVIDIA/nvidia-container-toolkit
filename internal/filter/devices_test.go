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
	log "github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestConstructor(t *testing.T) {

	logger, _ := testlog.NewNullLogger()

	device0 := discover.Device{
		Index:    "0",
		UUID:     "0",
		PCIBusID: discover.PCIBusID("0"),
	}
	device1 := discover.Device{
		Index:    "1",
		UUID:     "1",
		PCIBusID: discover.PCIBusID("1"),
	}
	device2 := discover.Device{
		Index:    "2",
		UUID:     "2",
		PCIBusID: discover.PCIBusID("2"),
	}
	device3 := discover.Device{
		Index:    "3",
		UUID:     "3",
		PCIBusID: discover.PCIBusID("3"),
	}
	controlDevice := discover.Device{
		UUID: "CONTROL",
	}
	mockDevices := []discover.Device{
		device0,
		device1,
		device2,
		device3,
		controlDevice,
	}
	d := discoverMock{
		devices: mockDevices,
	}

	var ok bool

	withDefaultLogger, ok := NewSelectDevicesFrom(d, "all", nil).(*devices)
	require.True(t, ok)
	require.Same(t, log.StandardLogger(), withDefaultLogger.logger)

	_, ok = NewSelectDevicesFromWithLogger(logger, d, "", nil).(*discover.None)
	require.True(t, ok)

	_, ok = NewSelectDevicesFromWithLogger(logger, d, "void", nil).(*discover.None)
	require.True(t, ok)

	_, ok = NewSelectDevicesFromWithLogger(logger, d, "none", nil).(*discover.None)
	require.True(t, ok)

	all, ok := NewSelectDevicesFromWithLogger(logger, d, "all", nil).(*devices)
	require.True(t, ok)

	devs, err := all.Devices()
	require.NoError(t, err)
	require.ElementsMatch(t, mockDevices, devs)

	f, ok := NewSelectDevicesFromWithLogger(logger, d, "0", nil).(*devices)
	require.True(t, ok)

	devs, err = f.Devices()
	require.NoError(t, err)
	require.Len(t, devs, 2)
	require.ElementsMatch(t, devs, []discover.Device{device0, controlDevice})

	f, ok = NewSelectDevicesFromWithLogger(logger, d, "0,2", nil).(*devices)
	require.True(t, ok)

	devs, err = f.Devices()
	require.NoError(t, err)
	require.Len(t, devs, 3)

	require.ElementsMatch(t, devs, []discover.Device{device0, device2, controlDevice})
}
