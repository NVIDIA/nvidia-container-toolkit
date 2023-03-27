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

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/proc/devices"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Interface is the interface for the system command
type Interface struct {
	logger *logrus.Logger
	dryRun bool

	nvidiaDevices nvidiaDevices
}

// New constructs a system command with the specified options
func New(opts ...Option) (*Interface, error) {
	i := &Interface{
		logger: logrus.StandardLogger(),
	}
	for _, opt := range opts {
		opt(i)
	}

	devices, err := devices.GetNVIDIADevices()
	if err != nil {
		return nil, fmt.Errorf("failed to create devices info: %v", err)
	}
	i.nvidiaDevices = nvidiaDevices{devices}

	return i, nil
}

// CreateNVIDIAControlDeviceNodesAt creates the NVIDIA control device nodes associated with the NVIDIA driver at the specified root.
func (m *Interface) CreateNVIDIAControlDeviceNodesAt(root string) error {
	controlNodes := []string{"/dev/nvidiactl", "/dev/nvidia-modeset", "/dev/nvidia-uvm", "/dev/nvidia-uvm-tools"}

	for _, node := range controlNodes {
		path := filepath.Join(root, node)
		err := m.CreateNVIDIADeviceNode(path)
		if err != nil {
			return fmt.Errorf("failed to create device node %s: %v", path, err)
		}
	}

	return nil
}

// CreateNVIDIADeviceNode creates a specified device node associated with the NVIDIA driver.
func (m *Interface) CreateNVIDIADeviceNode(path string) error {
	node := filepath.Base(path)
	if !strings.HasPrefix(node, "nvidia") {
		return fmt.Errorf("invalid device node %q", node)
	}

	major, err := m.nvidiaDevices.Major(node)
	if err != nil {
		return fmt.Errorf("failed to determine major: %v", err)
	}

	minor, err := m.nvidiaDevices.Minor(node)
	if err != nil {
		return fmt.Errorf("failed to determine minor: %v", err)
	}

	return m.createDeviceNode(path, int(major), int(minor))
}

func (m *Interface) createDeviceNode(path string, major int, minor int) error {
	if m.dryRun {
		m.logger.Infof("Running: mknod --mode=0666 %s c %d %d", path, major, minor)
		return nil
	}

	if _, err := os.Stat(path); err == nil {
		m.logger.Infof("Skipping: %s already exists", path)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %v", path, err)
	}

	err := unix.Mknod(path, unix.S_IFCHR, int(unix.Mkdev(uint32(major), uint32(minor))))
	if err != nil {
		return err
	}
	return unix.Chmod(path, 0666)
}

type nvidiaDevices struct {
	devices.Devices
}

// Major returns the major number for the specified NVIDIA device node.
// If the device node is not supported, an error is returned.
func (n *nvidiaDevices) Major(node string) (int64, error) {
	var valid bool
	var major devices.Major
	switch node {
	case "nvidia-uvm", "nvidia-uvm-tools":
		major, valid = n.Get(devices.NVIDIAUVM)
	case "nvidia-modeset", "nvidiactl":
		major, valid = n.Get(devices.NVIDIAGPU)
	}

	if !valid {
		return 0, fmt.Errorf("invalid device node %q", node)
	}

	return int64(major), nil
}

// Minor returns the minor number for the specified NVIDIA device node.
// If the device node is not supported, an error is returned.
func (n *nvidiaDevices) Minor(node string) (int64, error) {
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

	return 0, fmt.Errorf("invalid device node %q", node)
}
