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

package devices

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Device major numbers and device names for NVIDIA devices
const (
	NVIDIAUVMMinor      = 0
	NVIDIAUVMToolsMinor = 1
	NVIDIACTLMinor      = 255
	NVIDIAModesetMinor  = 254

	NVIDIAFrontend = Name("nvidia-frontend")
	NVIDIAGPU      = NVIDIAFrontend
	NVIDIACaps     = Name("nvidia-caps")
	NVIDIAUVM      = Name("nvidia-uvm")

	procDevicesPath    = "/proc/devices"
	nvidiaDevicePrefix = "nvidia"
)

// Name represents the name of a device as specified under /proc/devices
type Name string

// Major represents a device major as specified under /proc/devices
type Major int

// Devices represents the set of devices under /proc/devices
//
//go:generate moq -stub -out devices_mock.go . Devices
type Devices interface {
	Exists(Name) bool
	Get(Name) (Major, bool)
}

type devices map[Name]Major

var _ Devices = devices(nil)

// Exists checks if a Device with a given name exists or not
func (d devices) Exists(name Name) bool {
	_, exists := d[name]
	return exists
}

// Get a Device from Devices
func (d devices) Get(name Name) (Major, bool) {
	device, exists := d[name]
	return device, exists
}

// GetNVIDIADevices returns the set of NVIDIA Devices on the machine
func GetNVIDIADevices() (Devices, error) {
	devicesFile, err := os.Open(procDevicesPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error opening devices file: %v", err)
	}
	defer devicesFile.Close()

	return nvidiaDeviceFrom(devicesFile), nil
}

func nvidiaDeviceFrom(reader io.Reader) devices {
	allDevices := devicesFrom(reader)
	nvidiaDevices := make(devices)
	for n, d := range allDevices {
		if !strings.HasPrefix(string(n), nvidiaDevicePrefix) {
			continue
		}
		nvidiaDevices[n] = d
	}

	return nvidiaDevices
}

func devicesFrom(reader io.Reader) devices {
	allDevices := make(devices)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		device, major, err := processProcDeviceLine(scanner.Text())
		if err != nil {
			continue
		}
		allDevices[device] = major
	}
	return allDevices
}

func processProcDeviceLine(line string) (Name, Major, error) {
	trimmed := strings.TrimSpace(line)

	var name Name
	var major Major

	n, _ := fmt.Sscanf(trimmed, "%d %s", &major, &name)
	if n == 2 {
		return name, major, nil
	}

	return "", 0, fmt.Errorf("unparsable line: %v", line)
}
