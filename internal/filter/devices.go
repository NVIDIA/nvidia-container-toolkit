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
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	log "github.com/sirupsen/logrus"
)

const (
	visibleDevicesAll  = "all"
	visibleDevicesNone = "none"
	visibleDevicesVoid = "void"
)

type devices struct {
	discover.Discover
	logger   *log.Logger
	selector Selector
}

var _ discover.Discover = (*devices)(nil)

// NewSelectDevicesFrom creates a filter that selects devices based on the value of the
// visible devices string.
func NewSelectDevicesFrom(d discover.Discover, visibleDevices string, env EnvLookup) discover.Discover {
	return NewSelectDevicesFromWithLogger(log.StandardLogger(), d, visibleDevices, env)
}

// NewSelectDevicesFromWithLogger creates a filter as for NewSelectDevicesFrom with the
// specified logger.
func NewSelectDevicesFromWithLogger(logger *log.Logger, d discover.Discover, visibleDevices string, env EnvLookup) discover.Discover {
	if visibleDevices == "" || visibleDevices == visibleDevicesNone || visibleDevices == visibleDevicesVoid {
		return &discover.None{}
	}

	var visibleDeviceSelector Selector
	if visibleDevices == visibleDevicesAll {
		visibleDeviceSelector = StandardDevice()
	} else {
		visibleDeviceSelector = All(StandardDevice(), NewDeviceSelector(strings.Split(visibleDevices, ",")...))
	}

	controlDeviceIds := getControlDeviceIDsFromEnvWithLogger(logger, env)
	controlDeviceSelector := All(ControlDevice(), NewDeviceSelector(controlDeviceIds...))

	vd := devices{
		Discover: d,
		logger:   logger,
		selector: Any(visibleDeviceSelector, controlDeviceSelector),
	}

	return &vd
}

// Devices returns the list of selected devices after filtering based on the
// configured selector
func (d devices) Devices() ([]discover.Device, error) {
	devices, err := d.Discover.Devices()
	if err != nil {
		return nil, fmt.Errorf("error discovering devices: %v", err)
	}

	var selected []discover.Device
	for _, di := range devices {
		if d.selector.Selected(di) {
			d.logger.Infof("selecting device=%v", di)
			selected = append(selected, di)
		}
	}

	return selected, nil
}
