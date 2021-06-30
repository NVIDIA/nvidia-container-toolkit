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

package proc

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	procDevicesPath    = "/proc/devices"
	nvidiaDevicePrefix = "nvidia"
)

// Device represents a device as specified under /proc/devices
type Device struct {
	Name  string
	Major int
}

// NvidiaDevices represents the set of nvidia owned devices under /proc/devices
type NvidiaDevices interface {
	Exists(name string) bool
	Get(name string) (Device, bool)
}

type nvidiaDevices map[string]Device

var _ NvidiaDevices = nvidiaDevices(nil)

// Exists checks if a Device with a given name exists or not
func (d nvidiaDevices) Exists(name string) bool {
	_, exists := d[name]
	return exists
}

// Get a Device from NvidiaDevices
func (d nvidiaDevices) Get(name string) (Device, bool) {
	device, exists := d[name]
	return device, exists
}

func (d nvidiaDevices) add(devices ...Device) {
	for _, device := range devices {
		d[device.Name] = device
	}
}

// NewMockNvidiaDevices returns NvidiaDevices populated from the devices passed in
func NewMockNvidiaDevices(devices ...Device) NvidiaDevices {
	nvds := make(nvidiaDevices)
	nvds.add(devices...)
	return nvds
}

// GetNvidiaDevices returns the set of NvidiaDevices on the machine
func GetNvidiaDevices() (NvidiaDevices, error) {
	devicesFile, err := os.Open(procDevicesPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error opening devices file: %v", err)
	}
	defer devicesFile.Close()

	return processDeviceFile(devicesFile), nil
}

func processDeviceFile(devicesFile io.Reader) NvidiaDevices {
	nvidiaDevices := make(nvidiaDevices)
	scanner := bufio.NewScanner(devicesFile)
	for scanner.Scan() {
		device, major, err := processProcDeviceLine(scanner.Text())
		if err != nil {
			log.Printf("Skipping line in devices file: %v", err)
			continue
		}
		if strings.HasPrefix(device, nvidiaDevicePrefix) {
			nvidiaDevices.add(Device{device, major})
		}
	}
	return nvidiaDevices
}

func processProcDeviceLine(line string) (string, int, error) {
	trimmed := strings.TrimSpace(line)

	var name string
	var major int

	n, _ := fmt.Sscanf(trimmed, "%d %s", &major, &name)
	if n == 2 {
		return name, major, nil
	}

	return "", 0, fmt.Errorf("unparsable line: %v", line)
}
