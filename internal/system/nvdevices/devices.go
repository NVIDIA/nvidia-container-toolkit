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
	"path/filepath"

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
		i.mknoder = &mknodUnix{i.logger}
	}
	return i, nil
}

// createDeviceNode creates the specified device node with the require major and minor numbers.
// If a devRoot is configured, this is prepended to the path.
func (m *Interface) createDeviceNode(path string, major int, minor int) error {
	path = filepath.Join(m.devRoot, path)
	return m.Mknode(path, major, minor)
}
