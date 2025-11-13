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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
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
		i.mknoder = &mknodUnix{i.logger}
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

// CreateDevCharSymlinks creates symlinks at the specified path NVIDIA device nodes.
// If existingOnly is set to false, symlinks will be created for ALL possible devices.
func (m *Interface) CreateDevCharSymlinks(devCharPath string, existingOnly bool) error {
	if devCharPath == "" || devCharPath == "/" {
		return fmt.Errorf("invalid /dev/char path: %q", devCharPath)
	}
	lister, err := newAllPossible(m.logger, m.devRoot)
	if err != nil {
		return fmt.Errorf("failed to create all possible device lister: %v", err)
	}

	deviceNodes, err := lister.DeviceNodes()
	if err != nil {
		return fmt.Errorf("failed to get device nodes: %v", err)
	}

	var deviceNodeLocator lookup.Locator
	if existingOnly {
		deviceNodeLocator = lookup.NewCharDeviceLocator(
			lookup.WithLogger(m.logger),
			lookup.WithRoot(m.devRoot),
			lookup.WithCount(1),
			lookup.WithOptional(true),
		)
	} else {
		deviceNodeLocator = lookup.Always
	}

	var parentCreated bool
	for _, deviceNode := range deviceNodes {
		target := deviceNode.path
		// TODO: This assumes that the majors for the kernel modules align with
		// the majors for the actual device nodes.
		linkPath := filepath.Join(devCharPath, deviceNode.devCharName())

		candidates, err := deviceNodeLocator.Locate(target)
		if err != nil || len(candidates) == 0 {
			m.logger.Debugf("Ignoring non-existing device node %q", target)
		}

		m.logger.Infof("Creating link %s => %s", linkPath, target)
		if m.dryRun {
			continue
		}

		if !parentCreated {
			err := os.MkdirAll(devCharPath, 0755)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %v", devCharPath, err)
			}
			parentCreated = true
		}

		if err := os.Symlink(target, linkPath); err != nil {
			m.logger.Warningf("Could not create symlink: %v", err)
		}
	}

	return nil
}

// createDeviceNode creates the specified device node with the require major and minor numbers.
// If a devRoot is configured, this is prepended to the path.
func (m *Interface) createDeviceNode(path string, major int, minor int) error {
	path = filepath.Join(m.devRoot, path)
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
