/*
 * Copyright (c) NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nvpci

import (
	"fmt"
	"strings"
)

const (
	// PCIMellanoxVendorID represents PCI vendor id for Mellanox.
	PCIMellanoxVendorID uint16 = 0x15b3
	// PCINetworkControllerClass represents the PCI class for network controllers.
	PCINetworkControllerClass uint32 = 0x020000
	// PCIBridgeClass represents the PCI class for network controllers.
	PCIBridgeClass uint32 = 0x060400
)

// GetNetworkControllers returns all Mellanox Network Controller PCI devices on the system.
func (p *nvpci) GetNetworkControllers() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsNetworkController() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetPciBridges retrieves all Mellanox PCI(e) Bridges.
func (p *nvpci) GetPciBridges() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsPciBridge() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// IsNetworkController if class == 0x300.
func (d *NvidiaPCIDevice) IsNetworkController() bool {
	return d.Class == PCINetworkControllerClass
}

// IsPciBridge if class == 0x0604.
func (d *NvidiaPCIDevice) IsPciBridge() bool {
	return d.Class == PCIBridgeClass
}

// IsDPU returns if a device is a DPU.
func (d *NvidiaPCIDevice) IsDPU() bool {
	if !strings.Contains(d.DeviceName, "BlueField") {
		return false
	}
	// DPU is a multifunction device hence look only for the .0 function
	// and ignore subfunctions like .1, .2, etc.
	if strings.HasSuffix(d.Address, ".0") {
		return true
	}
	return false
}

// GetDPUs returns all Mellanox DPU devices on the system.
func (p *nvpci) GetDPUs() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetNetworkControllers()
	if err != nil {
		return nil, fmt.Errorf("error getting all network controllers: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsDPU() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}
