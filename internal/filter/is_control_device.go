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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	log "github.com/sirupsen/logrus"
)

const (
	devicesAll = "all"
)

type controlDevices struct {
	discover.Discover
	logger   *log.Logger
	selector Selector
}

var _ discover.Discover = (*controlDevices)(nil)

// NewControlDevicesFrom creates a filter that selects devices based on the value of the
// visible devices string.
func NewControlDevicesFrom(d discover.Discover, env EnvLookup) Selector {
	return NewControlDevicesFromWithLogger(log.StandardLogger(), d, env)
}

// NewControlDevicesFromWithLogger creates a filter as for NewControlDevicesFrom with the
// specified logger.
func NewControlDevicesFromWithLogger(logger *log.Logger, d discover.Discover, env EnvLookup) Selector {
	controlDevices := getControlDeviceIDsFromEnvWithLogger(logger, env)
	return NewDeviceSelector(controlDevices...)
}

type controlDevice struct{}

// ControlDevice returns a selector for control devices
func ControlDevice() Selector {
	return &controlDevice{}
}

// Selected returns true for a controll devices and false for standard devices. A control device
// has an empty index and PCI bus ID and a non-empty UUID.
func (s controlDevice) Selected(device discover.Device) bool {
	if device.Index != "" {
		return false
	}
	if device.PCIBusID != "" {
		return false
	}
	if device.UUID == "" {
		return false
	}

	return true
}

func getControlDeviceIDsFromEnvWithLogger(logger *log.Logger, env EnvLookup) []string {
	controlDevices := []string{discover.ControlDeviceUUID}

	migControlDevices := getMIGControlDevicesFromEnvWithLogger(logger, env)

	return append(controlDevices, migControlDevices...)
}

func getMIGControlDevicesFromEnvWithLogger(logger *log.Logger, env EnvLookup) []string {
	if env == nil {
		logger.Debugf("Environment not specified; no MIG Control devices selected")
		return []string{}
	}

	var controlDevices []string

	// Add MIG control devices
	migEnvUUIDMap := map[string]string{
		discover.MIGConfigDeviceUUID:  "NVIDIA_MIG_CONFIG_DEVICES",
		discover.MIGMonitorDeviceUUID: "NVIDIA_MIG_MONITOR_DEVICES",
	}
	for uuid, migEnv := range migEnvUUIDMap {
		config, exists := env.LookupEnv(migEnv)
		if !exists {
			logger.Debugf("Envvar %v not set", migEnv)
			continue
		}
		if config == devicesAll {
			logger.Infof("Found %v=%v; selecting MIG %v devices", migEnv, config, uuid)
			controlDevices = append(controlDevices, uuid)
		} else {
			logger.Debugf("Found %v=%v; Skipping MIG %v devices (%v != %v)", migEnv, config, uuid, config, devicesAll)
		}
	}
	return controlDevices
}
