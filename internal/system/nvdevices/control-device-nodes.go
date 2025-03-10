/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package nvdevices

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
)

// A controlDeviceNode represents an NVIDIA devices node for control or meta devices.
// Such device nodes are typically required regardless of which GPU is being accessed.
type controlDeviceNode string

func (c controlDeviceNode) path() string {
	return filepath.Join("dev", string(c))
}

// CreateNVIDIAControlDevices creates the NVIDIA control device nodes at the configured devRoot.
func (m *Interface) CreateNVIDIAControlDevices() error {
	controlNodes := []controlDeviceNode{"nvidiactl", "nvidia-modeset", "nvidia-uvm", "nvidia-uvm-tools"}
	for _, node := range controlNodes {
		if err := m.createControlDeviceNode(node); err != nil {
			return fmt.Errorf("failed to create device node %s: %w", node, err)
		}
	}
	return nil
}

// createControlDeviceNode creates the specified NVIDIA device node at the configured devRoot.
func (m *Interface) createControlDeviceNode(node controlDeviceNode) error {
	if !strings.HasPrefix(string(node), "nvidia") {
		return fmt.Errorf("invalid device node %q: %w", node, errInvalidDeviceNode)
	}

	major, err := m.controlDeviceNodeMajor(node)
	if err != nil {
		return fmt.Errorf("failed to determine major: %w", err)
	}

	minor, err := m.controlDeviceNodeMinor(node)
	if err != nil {
		return fmt.Errorf("failed to determine minor: %w", err)
	}

	return m.createDeviceNode(node.path(), int(major), int(minor))
}

// controlDeviceNodeMajor returns the major number for the specified NVIDIA control device node.
// If the device node is not supported, an error is returned.
func (m *Interface) controlDeviceNodeMajor(node controlDeviceNode) (int64, error) {
	var valid bool
	var major devices.Major
	switch node {
	case "nvidia-uvm", "nvidia-uvm-tools":
		major, valid = m.Get(devices.NVIDIAUVM)
	case "nvidia-modeset", "nvidiactl":
		major, valid = m.Get(devices.NVIDIAGPU)
	}

	if valid {
		return int64(major), nil
	}

	return 0, errInvalidDeviceNode
}

// controlDeviceNodeMinor returns the minor number for the specified NVIDIA control device node.
// If the device node is not supported, an error is returned.
func (m *Interface) controlDeviceNodeMinor(node controlDeviceNode) (int64, error) {
	switch node {
	case "nvidia-modeset":
		return devices.NVIDIAModesetMinor, nil
	case "nvidia-uvm-tools":
		return devices.NVIDIAUVMToolsMinor, nil
	case "nvidia-uvm":
		return devices.NVIDIAUVMMinor, nil
	case "nvidiactl":
		return devices.NVIDIACTLMinor, nil
	}

	return 0, errInvalidDeviceNode
}
