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
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

type nodeLister interface {
	DeviceNodes() ([]deviceNode, error)
}

type existing struct {
	logger  logger.Interface
	devRoot string
}

// DeviceNodes returns a list of NVIDIA device nodes in the specified root.
// The nvidia-nvswitch* and nvidia-nvlink devices are excluded.
func (m existing) DeviceNodes() ([]deviceNode, error) {
	locator := lookup.NewCharDeviceLocator(
		lookup.WithLogger(m.logger),
		lookup.WithRoot(m.devRoot),
		lookup.WithOptional(true),
	)

	devices, err := locator.Locate("/dev/nvidia*")
	if err != nil {
		m.logger.Warningf("Error while locating device: %v", err)
	}

	capDevices, err := locator.Locate("/dev/nvidia-caps/nvidia-*")
	if err != nil {
		m.logger.Warningf("Error while locating caps device: %v", err)
	}

	if len(devices) == 0 && len(capDevices) == 0 {
		m.logger.Infof("No NVIDIA devices found in %s", m.devRoot)
		return nil, nil
	}

	var deviceNodes []deviceNode
	for _, d := range append(devices, capDevices...) {
		if m.nodeIsBlocked(d) {
			continue
		}
		var stat unix.Stat_t
		err := unix.Stat(d, &stat)
		if err != nil {
			m.logger.Warningf("Could not stat device: %v", err)
			continue
		}
		deviceNodes = append(deviceNodes, newDeviceNode(d, stat))
	}

	return deviceNodes, nil
}

// nodeIsBlocked returns true if the specified device node should be ignored.
func (m existing) nodeIsBlocked(path string) bool {
	blockedPrefixes := []string{"nvidia-fs", "nvidia-nvswitch", "nvidia-nvlink"}
	nodeName := filepath.Base(path)
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(nodeName, prefix) {
			return true
		}
	}
	return false
}
