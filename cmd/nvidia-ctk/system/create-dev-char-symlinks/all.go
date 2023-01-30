/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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
**/

package devchar

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvpci"
)

type allPossible struct {
	logger       *logrus.Logger
	driverRoot   string
	deviceMajors devices.Devices
	migCaps      nvcaps.MigCaps
}

// newAllPossible returns a new allPossible device node lister.
// This lister lists all possible device nodes for NVIDIA GPUs, control devices, and capability devices.
func newAllPossible(logger *logrus.Logger, driverRoot string) (nodeLister, error) {
	deviceMajors, err := devices.GetNVIDIADevices()
	if err != nil {
		return nil, fmt.Errorf("failed reading device majors: %v", err)
	}
	migCaps, err := nvcaps.NewMigCaps()
	if err != nil {
		return nil, fmt.Errorf("failed to read MIG caps: %v", err)
	}
	if migCaps == nil {
		migCaps = make(nvcaps.MigCaps)
	}

	l := allPossible{
		logger:       logger,
		driverRoot:   driverRoot,
		deviceMajors: deviceMajors,
		migCaps:      migCaps,
	}

	return l, nil
}

// DeviceNodes returns a list of all possible device nodes for NVIDIA GPUs, control devices, and capability devices.
func (m allPossible) DeviceNodes() ([]deviceNode, error) {
	gpus, err := nvpci.NewFrom(
		filepath.Join(m.driverRoot, nvpci.PCIDevicesRoot),
	).GetGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU information: %v", err)
	}

	count := len(gpus)
	if count == 0 {
		m.logger.Infof("No NVIDIA devices found in %s", m.driverRoot)
		return nil, nil
	}

	deviceNodes, err := m.getControlDeviceNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get control device nodes: %v", err)
	}

	for gpu := 0; gpu < count; gpu++ {
		deviceNodes = append(deviceNodes, m.getGPUDeviceNodes(gpu)...)
		deviceNodes = append(deviceNodes, m.getNVCapDeviceNodes(gpu)...)
	}

	return deviceNodes, nil
}

// getControlDeviceNodes generates a list of control devices
func (m allPossible) getControlDeviceNodes() ([]deviceNode, error) {
	var deviceNodes []deviceNode

	// Define the control devices for standard GPUs.
	controlDevices := []deviceNode{
		m.newDeviceNode(devices.NVIDIAGPU, "/dev/nvidia-modeset", devices.NVIDIAModesetMinor),
		m.newDeviceNode(devices.NVIDIAGPU, "/dev/nvidiactl", devices.NVIDIACTLMinor),
		m.newDeviceNode(devices.NVIDIAUVM, "/dev/nvidia-uvm", devices.NVIDIAUVMMinor),
		m.newDeviceNode(devices.NVIDIAUVM, "/dev/nvidia-uvm-tools", devices.NVIDIAUVMToolsMinor),
	}

	deviceNodes = append(deviceNodes, controlDevices...)

	for _, migControlDevice := range []nvcaps.MigCap{"config", "monitor"} {
		migControlMinor, exist := m.migCaps[migControlDevice]
		if !exist {
			continue
		}

		d := m.newDeviceNode(
			devices.NVIDIACaps,
			migControlMinor.DevicePath(),
			int(migControlMinor),
		)

		deviceNodes = append(deviceNodes, d)
	}

	return deviceNodes, nil
}

// getGPUDeviceNodes generates a list of device nodes for a given GPU.
func (m allPossible) getGPUDeviceNodes(gpu int) []deviceNode {
	d := m.newDeviceNode(
		devices.NVIDIAGPU,
		fmt.Sprintf("/dev/nvidia%d", gpu),
		gpu,
	)

	return []deviceNode{d}
}

// getNVCapDeviceNodes generates a list of cap device nodes for a given GPU.
func (m allPossible) getNVCapDeviceNodes(gpu int) []deviceNode {
	var selectedCapMinors []nvcaps.MigMinor
	for gi := 0; ; gi++ {
		giCap := nvcaps.NewGPUInstanceCap(gpu, gi)
		giMinor, exist := m.migCaps[giCap]
		if !exist {
			break
		}
		selectedCapMinors = append(selectedCapMinors, giMinor)
		for ci := 0; ; ci++ {
			ciCap := nvcaps.NewComputeInstanceCap(gpu, gi, ci)
			ciMinor, exist := m.migCaps[ciCap]
			if !exist {
				break
			}
			selectedCapMinors = append(selectedCapMinors, ciMinor)
		}
	}

	var deviceNodes []deviceNode
	for _, capMinor := range selectedCapMinors {
		d := m.newDeviceNode(
			devices.NVIDIACaps,
			capMinor.DevicePath(),
			int(capMinor),
		)
		deviceNodes = append(deviceNodes, d)
	}

	return deviceNodes
}

// newDeviceNode creates a new device node with the specified path and major/minor numbers.
// The path is adjusted for the specified driver root.
func (m allPossible) newDeviceNode(deviceName devices.Name, path string, minor int) deviceNode {
	major, _ := m.deviceMajors.Get(deviceName)

	return deviceNode{
		path:  filepath.Join(m.driverRoot, path),
		major: uint32(major),
		minor: uint32(minor),
	}
}
