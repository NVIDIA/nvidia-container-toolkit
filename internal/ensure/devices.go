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

package ensure

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	log "github.com/sirupsen/logrus"
)

type ensureDevices struct {
	logger *log.Logger
	discover.Discover
	lookup lookup.Locator
	root   string
}

// NewEnsureDevices creates a discoverer that wraps the specified discoverer and ensures that the
// device nodes for the discoverer are created. If a root is specified, the device nodes
// rooted there are also created.
func NewEnsureDevices(d discover.Discover, root string) discover.Discover {
	return NewEnsureDevicesWithLogger(log.StandardLogger(), d, root)
}

// NewEnsureDevicesWithLogger creates a discoverer that wraps the specified discoverer and ensures that the
// device nodes for the discoverer are created. If a root is specified, the device nodes
// rooted there are also created. The specified logger is used.
func NewEnsureDevicesWithLogger(logger *log.Logger, d discover.Discover, root string) discover.Discover {
	e := ensureDevices{
		Discover: d,
		logger:   logger,
		lookup:   lookup.NewPathLocatorWithLogger(logger, root),
		root:     root,
	}
	return &e
}

func (d ensureDevices) Devices() ([]discover.Device, error) {
	devices, err := d.Discover.Devices()
	if err != nil {
		return nil, fmt.Errorf("error discovering devices: %v", err)
	}
	for _, di := range devices {
		for _, dn := range di.DeviceNodes {
			d.deviceNode(dn)
		}
	}
	return devices, nil
}

func (d ensureDevices) deviceNode(dn discover.DeviceNode) error {
	err := d.device(dn.Path, dn.Major, dn.Minor)
	if err != nil {
		d.logger.Errorf("Error creating device node %+v: %v", dn, err)
	}

	if d.root != "" && d.root != "/" {
		rootedPath := discover.DevicePath(filepath.Join(d.root, string(dn.Path)))
		err = d.device(rootedPath, dn.Major, dn.Minor)
		if err != nil {
			d.logger.Errorf("Error creating device node %+v: %v", dn, err)
		}
	}
	return nil
}

func (d ensureDevices) device(path discover.DevicePath, major int, minor int) error {
	// TODO: We may want to check that the device node has the required permissions
	_, err := os.Stat(string(path))
	if err == nil {
		d.logger.Infof("Device node %v already exists", path)
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		d.logger.Errorf("Error getting info for device node %v: %v", path, err)
		return fmt.Errorf("error getting device node info: %w", err)
	}
	// See: https://docs.nvidia.com/cuda/cuda-installation-guide-linux/index.html#runfile-verifications

	// TODO: We should use nvidia-modprobe or a tool based off that instead
	args := []string{
		"-m", "666",
		string(path), "c",
		fmt.Sprint(major),
		fmt.Sprint(minor),
	}

	return d.run("mknod", args...)
}

func (d ensureDevices) run(cmd string, args ...string) error {
	paths, err := d.lookup.Locate(cmd)
	if err != nil {
		return fmt.Errorf("error finding command %v: %v", cmd, err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("command %v not found in path", cmd)
	}
	path := paths[0]

	d.logger.Debugf("Running %v", append([]string{path}, args...))
	err = exec.Command(path, args...).Run()
	if err != nil {
		return fmt.Errorf("error running %v: %v", append([]string{path}, args...), err)
	}

	return nil
}
