/**
# Copyright (c) NVIDIA CORPORATIOm.  All rights reserved.
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

var errInvalidDeviceNode = errors.New("invalid device node")

// Interface provides a set of utilities for interacting with NVIDIA devices on the system.
type Interface struct {
	devices.Devices

	logger logger.Interface

	dryRun bool
	// devRoot is the root directory where device nodes are expected to exist.
	devRoot string

	mknoder
}

// New constructs a new Interface struct with the specified options.
func New(opts ...Option) (*Interface, error) {
	i := &Interface{}
	for _, opt := range opts {
		opt(i)
	}

	if i.logger == nil {
		i.logger = logger.New()
	}
	if i.devRoot == "" {
		i.devRoot = "/"
	}
	if i.Devices == nil {
		devices, err := devices.GetNVIDIADevices()
		if err != nil {
			return nil, fmt.Errorf("failed to create devices info: %v", err)
		}
		i.Devices = devices
	}

	if i.dryRun {
		i.mknoder = &mknodLogger{i.logger}
	} else {
		i.mknoder = &mknodUnix{}
	}
	return i, nil
}

// CreateNVIDIAControlDevices creates the NVIDIA control device nodes at the configured devRoot.
func (m *Interface) CreateNVIDIAControlDevices() error {
	controlNodes := []string{"nvidiactl", "nvidia-modeset", "nvidia-uvm", "nvidia-uvm-tools"}
	for _, node := range controlNodes {
		err := m.CreateNVIDIADevice(node)
		if err != nil {
			return fmt.Errorf("failed to create device node %s: %w", node, err)
		}
	}
	return nil
}

// CreateNVIDIADevice creates the specified NVIDIA device node at the configured devRoot.
func (m *Interface) CreateNVIDIADevice(node string) error {
	node = filepath.Base(node)
	if !strings.HasPrefix(node, "nvidia") {
		return fmt.Errorf("invalid device node %q: %w", node, errInvalidDeviceNode)
	}

	major, err := m.Major(node)
	if err != nil {
		return fmt.Errorf("failed to determine major: %w", err)
	}

	minor, err := m.Minor(node)
	if err != nil {
		return fmt.Errorf("failed to determine minor: %w", err)
	}

	return m.createDeviceNode(filepath.Join("dev", node), int(major), int(minor))
}

// createDeviceNode creates the specified device node with the require major and minor numbers.
// If a devRoot is configured, this is prepended to the path.
func (m *Interface) createDeviceNode(path string, major int, minor int) error {
	path = filepath.Join(m.devRoot, path)
	if _, err := os.Stat(path); err == nil {
		m.logger.Infof("Skipping: %s already exists", path)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %v", path, err)
	}

	return m.Mknode(path, major, minor)
}

// Major returns the major number for the specified NVIDIA device node.
// If the device node is not supported, an error is returned.
func (m *Interface) Major(node string) (int64, error) {
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

// Minor returns the minor number for the specified NVIDIA device node.
// If the device node is not supported, an error is returned.
func (m *Interface) Minor(node string) (int64, error) {
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
