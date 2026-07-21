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

package nvcdi

import (
	"fmt"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
)

type migCapsLib nvcdilib

// migCapDeviceSpecGenerator generates the CDI device spec for a single MIG
// management capability (config or monitor).
type migCapDeviceSpecGenerator struct {
	lib *migCapsLib
	cap nvcaps.MigCap
}

var _ deviceSpecGeneratorFactory = (*migCapsLib)(nil)

// GetCommonEdits returns an empty set of edits for MIG capability devices.
func (l *migCapsLib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return l.editsFactory.FromDiscoverer(discover.None{})
}

// DeviceSpecGenerators returns the CDI device spec generators for the specified
// MIG management capabilities.
// Valid IDs are 'config' and 'monitor'.
func (l *migCapsLib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	var deviceSpecGenerators DeviceSpecGenerators
	for _, id := range ids {
		cap := nvcaps.MigCap(id)
		switch cap {
		case "config", "monitor":
			deviceSpecGenerators = append(deviceSpecGenerators, &migCapDeviceSpecGenerator{lib: l, cap: cap})
		default:
			return nil, fmt.Errorf("invalid MIG capability %q: must be one of [config, monitor]", id)
		}
	}
	return deviceSpecGenerators, nil
}

// GetDeviceSpecs returns the CDI device specs for the MIG management capability.
func (g *migCapDeviceSpecGenerator) GetDeviceSpecs() ([]specs.Device, error) {
	l := g.lib

	migCaps, err := nvcaps.NewMigCapsFromRoot(l.driver.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to get MIG capability device paths: %w", err)
	}
	if migCaps == nil {
		return nil, fmt.Errorf("cannot inject MIG %s capability: system is not MIG capable", g.cap)
	}

	devicePath, err := migCaps.GetCapDevicePath(g.cap)
	if err != nil {
		return nil, fmt.Errorf("failed to get device path for MIG %s capability: %w", g.cap, err)
	}

	deviceNodes := discover.NewCharDeviceDiscoverer(
		l.logger,
		l.driver.DevRoot,
		[]string{devicePath},
	)

	// The MIG capability nodes are nested under /dev/nvidia-caps. Add a hook to
	// set the permissions of the parent folder so that non-root users in the
	// container can access the injected node.
	folderPermissionHooks := (*nvcdilib)(l).newDeviceFolderPermissionHookDiscoverer(deviceNodes)

	edits, err := l.editsFactory.FromDiscoverer(discover.Merge(deviceNodes, folderPermissionHooks))
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for MIG %s capability: %w", g.cap, err)
	}

	deviceSpec := specs.Device{
		Name:           string(g.cap),
		ContainerEdits: *edits.ContainerEdits,
	}
	return []specs.Device{deviceSpec}, nil
}
