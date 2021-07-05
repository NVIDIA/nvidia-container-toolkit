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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvml"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/proc"
	log "github.com/sirupsen/logrus"
)

const (
	// ControlDeviceUUID is used as the UUID for control devices such as nvidiactl or nvidia-modeset
	ControlDeviceUUID = "CONTROL"

	// MIGConfigDeviceUUID is used to indicate the MIG config control device
	MIGConfigDeviceUUID = "CONFIG"

	// MIGMonitorDeviceUUID is used to indicate the MIG monitor control device
	MIGMonitorDeviceUUID = "MONITOR"

	nvidiaGPUDeviceName  = "nvidia-frontend"
	nvidiaCapsDeviceName = "nvidia-caps"
	nvidiaUVMDeviceName  = "nvidia-uvm"
)

type nvmlDiscover struct {
	None
	logger        *log.Logger
	nvml          nvml.Interface
	migCaps       map[ProcPath]DeviceNode
	nvidiaDevices proc.NvidiaDevices
}

var _ Discover = (*nvmlDiscover)(nil)

// NewNVMLDiscover constructs a discoverer that uses NVML to find the devices
// available on a system.
func NewNVMLDiscover(nvml nvml.Interface) (Discover, error) {
	return NewNVMLDiscoverWithLogger(log.StandardLogger(), nvml)
}

// NewNVMLDiscoverWithLogger constructs a discovered as with NewNVMLDiscover with the specified
// logger
func NewNVMLDiscoverWithLogger(logger *log.Logger, nvml nvml.Interface) (Discover, error) {
	nvidiaDevices, err := proc.GetNvidiaDevices()
	if err != nil {
		return nil, fmt.Errorf("error loading NVIDIA devices: %v", err)
	}

	var migCaps map[ProcPath]DeviceNode
	nvcapsDevice, exists := nvidiaDevices.Get(nvidiaCapsDeviceName)
	if !exists {
		logger.Warnf("%v nvcaps device could not be found", nvidiaCapsDeviceName)
	} else if migCaps, err = getMigCaps(nvcapsDevice.Major); err != nil {
		logger.Warnf("Could not load MIG capability devices: %v", err)
		migCaps = nil
	}

	discover := &nvmlDiscover{
		logger:        logger,
		nvml:          nvml,
		migCaps:       migCaps,
		nvidiaDevices: nvidiaDevices,
	}

	return discover, nil
}

// hasMigSupport checks if MIG device discovery is supported.
// Cases where this will be disabled include where no MIG minors file is
// present.
func (d nvmlDiscover) hasMigSupport() bool {
	return len(d.migCaps) > 0
}

func (d *nvmlDiscover) Devices() ([]Device, error) {
	ret := d.nvml.Init()
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error initalizing NVML: %v", ret.Error())
	}
	defer d.tryShutdownNVML()

	c, ret := d.nvml.DeviceGetCount()
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting device count: %v", ret.Error())
	}

	var handles []nvml.Device
	for i := 0; i < c; i++ {
		handle, ret := d.nvml.DeviceGetHandleByIndex(i)
		if ret.Value() != nvml.SUCCESS {
			return nil, fmt.Errorf("error getting device handle for device %v: %v", i, ret.Error())
		}

		if !d.hasMigSupport() {
			handles = append(handles, handle)
			continue
		}

		migHandles, err := getMIGHandlesForDevice(handle)
		if err != nil {
			return nil, fmt.Errorf("error getting MIG handles for device: %v", err)
		}
		if len(migHandles) == 0 {
			handles = append(handles, handle)
		}
		handles = append(handles, migHandles...)
	}

	return d.devicesByHandle(handles)
}

func (d *nvmlDiscover) devicesByHandle(handles []nvml.Device) ([]Device, error) {
	var devices []Device
	var largestMinorNumber int
	for _, h := range handles {
		device, err := d.deviceFromNVMLHandle(h)
		if err != nil {
			return nil, fmt.Errorf("error constructing device from handle %v: %v", h, err)
		}
		devices = append(devices, device)

		if largestMinorNumber < device.DeviceNodes[0].Minor {
			largestMinorNumber = device.DeviceNodes[0].Minor
		}
	}

	controlDevices, err := d.getControlDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting control devices: %v", err)
	}
	devices = append(devices, controlDevices...)

	if d.hasMigSupport() {
		migControlDevices, err := d.getMigControlDevices(largestMinorNumber)
		if err != nil {
			return nil, fmt.Errorf("error getting MIG control devices: %v", err)
		}
		devices = append(devices, migControlDevices...)
	}

	return devices, nil
}

func (d *nvmlDiscover) deviceFromNVMLHandle(handle nvml.Device) (Device, error) {
	if d.hasMigSupport() {
		isMigDevice, ret := handle.IsMigDeviceHandle()
		if ret.Value() != nvml.SUCCESS {
			return Device{}, fmt.Errorf("error checking device handle: %v", ret.Error())
		}

		if isMigDevice {
			return d.deviceFromMIGDeviceHandle(handle)
		}
	}

	return d.deviceFromFullDeviceHandle(handle)
}

func (d *nvmlDiscover) deviceFromFullDeviceHandle(handle nvml.Device) (Device, error) {
	index, ret := handle.GetIndex()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting device index: %v", ret.Error())
	}

	uuid, ret := handle.GetUUID()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting device UUID: %v", ret.Error())
	}

	pciInfo, ret := handle.GetPciInfo()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting PCI info: %v", ret.Error())
	}
	busID := NewPCIBusID(pciInfo)

	minor, ret := handle.GetMinorNumber()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting minor number: %v", ret.Error())
	}

	nvidiaGPUDevice, exists := d.nvidiaDevices.Get(nvidiaGPUDeviceName)
	if !exists {
		return Device{}, fmt.Errorf("device for '%v' does not exist", nvidiaGPUDeviceName)
	}

	deviceNode := DeviceNode{
		Path:  DevicePath(fmt.Sprintf("/dev/nvidia%d", minor)),
		Major: nvidiaGPUDevice.Major,
		Minor: minor,
	}

	device := Device{
		Index:       fmt.Sprintf("%d", index),
		PCIBusID:    busID,
		UUID:        uuid,
		DeviceNodes: []DeviceNode{deviceNode},
		ProcPaths:   []ProcPath{busID.GetProcPath()},
	}

	return device, nil
}

func (d *nvmlDiscover) deviceFromMIGDeviceHandle(handle nvml.Device) (Device, error) {
	parent, ret := handle.GetDeviceHandleFromMigDeviceHandle()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting parent device handle: %v", ret.Error())
	}

	gpu, ret := parent.GetMinorNumber()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting GPU minor number: %v", ret.Error())
	}

	parentDevice, err := d.deviceFromFullDeviceHandle(parent)
	if err != nil {
		return Device{}, fmt.Errorf("error getting parent device: %v", err)
	}

	index, ret := handle.GetIndex()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting device index: %v", ret.Error())
	}

	uuid, ret := handle.GetUUID()
	if ret.Value() != nvml.SUCCESS {
		return Device{}, fmt.Errorf("error getting device UUID: %v", ret.Error())
	}

	capDeviceNodes := []DeviceNode{}
	procPaths, err := getProcPathsForMigDevice(gpu, handle)
	if err != nil {
		return Device{}, fmt.Errorf("error getting proc paths for MIG device: %v", err)
	}

	for _, p := range procPaths {
		capDeviceNode, ok := d.migCaps[p]
		if !ok {
			return Device{}, fmt.Errorf("could not determine cap device path for %v", p)
		}
		capDeviceNodes = append(capDeviceNodes, capDeviceNode)
	}

	device := Device{
		Index:       fmt.Sprintf("%s:%d", parentDevice.Index, index),
		UUID:        uuid,
		DeviceNodes: append(parentDevice.DeviceNodes, capDeviceNodes...),
		ProcPaths:   append(parentDevice.ProcPaths, procPaths...),
	}

	return device, nil
}

func (d *nvmlDiscover) getControlDevices() ([]Device, error) {
	devices := []struct {
		name  string
		path  string
		minor int
	}{
		// TODO: Where is the best place to find these device Minors programatically?
		{nvidiaGPUDeviceName, "/dev/nvidia-modeset", 254},
		{nvidiaGPUDeviceName, "/dev/nvidiactl", 255},
		{nvidiaUVMDeviceName, "/dev/nvidia-uvm", 0},
		{nvidiaUVMDeviceName, "/dev/nvidia-uvm-tools", 1},
	}

	var controlDevices []Device
	for _, info := range devices {
		device, exists := d.nvidiaDevices.Get(info.name)
		if !exists {
			d.logger.Warnf("device name %v not defined; skipping control devices %v", info.name, info.path)
			continue
		}

		deviceNode := DeviceNode{
			Path:  DevicePath(info.path),
			Major: device.Major,
			Minor: info.minor,
		}

		controlDevices = append(controlDevices, Device{
			UUID:        ControlDeviceUUID,
			DeviceNodes: []DeviceNode{deviceNode},
			ProcPaths:   []ProcPath{},
		})

	}

	return controlDevices, nil
}

func (d *nvmlDiscover) getMigControlDevices(numGpus int) ([]Device, error) {
	targets := map[string]ProcPath{
		MIGConfigDeviceUUID:  ProcPath("/proc/driver/nvidia/capabilities/mig/config"),
		MIGMonitorDeviceUUID: ProcPath("/proc/driver/nvidia/capabilities/mig/monitor"),
	}

	var devices []Device
	for id, procPath := range targets {
		deviceNode, exists := d.migCaps[procPath]
		if !exists {
			return nil, fmt.Errorf("device node for '%v' is undefined", procPath)
		}

		var procPaths []ProcPath
		for gpu := 0; gpu <= numGpus; gpu++ {
			procPaths = append(procPaths, ProcPath(fmt.Sprintf("/proc/driver/nvidia/capabilities/gpu%d/mig", gpu)))
		}
		procPaths = append(procPaths, procPath)

		devices = append(devices, Device{
			UUID:        id,
			DeviceNodes: []DeviceNode{deviceNode},
			ProcPaths:   procPaths,
		})
	}

	return devices, nil
}

func getProcPathsForMigDevice(gpu int, handle nvml.Device) ([]ProcPath, error) {
	gi, ret := handle.GetGPUInstanceId()
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting GPU instance ID: %v", ret.Error())
	}

	ci, ret := handle.GetComputeInstanceId()
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting comput instance ID: %v", ret.Error())
	}

	procPaths := []ProcPath{
		ProcPath(fmt.Sprintf("/proc/driver/nvidia/capabilities/gpu%d/mig/gi%d/access", gpu, gi)),
		ProcPath(fmt.Sprintf("/proc/driver/nvidia/capabilities/gpu%d/mig/gi%d/ci%d/access", gpu, gi, ci)),
	}

	return procPaths, nil
}

func getMIGHandlesForDevice(handle nvml.Device) ([]nvml.Device, error) {
	currentMigMode, _, ret := handle.GetMigMode()
	if ret.Value() == nvml.ERROR_NOT_SUPPORTED {
		return nil, nil
	}
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting MIG mode for device: %v", ret.Error())
	}
	if currentMigMode == nvml.DEVICE_MIG_DISABLE {
		return nil, nil
	}

	maxMigDeviceCount, ret := handle.GetMaxMigDeviceCount()
	if ret.Value() != nvml.SUCCESS {
		return nil, fmt.Errorf("error getting number of MIG devices: %v", ret.Error())
	}

	var migHandles []nvml.Device
	for mi := 0; mi < maxMigDeviceCount; mi++ {
		migHandle, ret := handle.GetMigDeviceHandleByIndex(mi)
		if ret.Value() == nvml.ERROR_NOT_FOUND {
			continue
		}

		if ret.Value() != nvml.SUCCESS {
			return nil, fmt.Errorf("error getting MIG device %v: %v", mi, ret.Error())
		}

		migHandles = append(migHandles, migHandle)
	}

	return migHandles, nil
}

func (d *nvmlDiscover) tryShutdownNVML() {
	ret := d.nvml.Shutdown()
	if ret.Value() != nvml.SUCCESS {
		d.logger.Warnf("Could not shut down NVML: %v", ret.Error())
	}
}

// NewPCIBusID provides a utility function that returns the string representation
// of the bus ID.
func NewPCIBusID(p nvml.PciInfo) PCIBusID {
	var bytes []byte
	for _, b := range p.BusId {
		if byte(b) == '\x00' {
			break
		}
		bytes = append(bytes, byte(b))
	}
	return PCIBusID(string(bytes))
}

// GetProcPath returns the path in /proc associated with the PCI bus ID
func (p PCIBusID) GetProcPath() ProcPath {
	id := strings.ToLower(p.String())

	if strings.HasPrefix(id, "0000") {
		id = id[4:]
	}
	return ProcPath(filepath.Join("/proc/driver/nvidia/gpus", id))
}

func (p PCIBusID) String() string {
	return string(p)
}
