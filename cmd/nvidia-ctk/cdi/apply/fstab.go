/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package apply

import (
	"fmt"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"
)

// FstabEntry represents a single entry in an fstab file.
type FstabEntry struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       string
	Pass       string
}

// String returns the fstab entry as a formatted string.
func (e FstabEntry) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
		e.Device, e.MountPoint, e.FSType, e.Options, e.Dump, e.Pass)
}

// Mount is a wrapper around specs.Mount for fstab conversion.
type Mount struct {
	*specs.Mount
}

// toFstab converts a CDI mount specification to an fstab entry.
func (m *Mount) toFstab() FstabEntry {
	entry := FstabEntry{
		Dump: "0",
		Pass: "0",
	}

	// Device/source (HostPath)
	entry.Device = m.HostPath
	if entry.Device == "" {
		entry.Device = "none"
	}

	// Mount point (ContainerPath)
	entry.MountPoint = m.ContainerPath

	// Filesystem type
	entry.FSType = m.Type
	if entry.FSType == "" {
		entry.FSType = "none"
	}

	// Mount options
	entry.Options = "defaults"
	if len(m.Options) > 0 {
		entry.Options = strings.Join(m.Options, ",")
	}

	return entry
}

// DeviceNode is a wrapper around specs.DeviceNode for fstab conversion.
type DeviceNode struct {
	*specs.DeviceNode
}

// toFstab converts a CDI device node specification to an fstab bind mount entry.
func (d *DeviceNode) toFstab() FstabEntry {
	entry := FstabEntry{
		FSType: "none",
		Dump:   "0",
		Pass:   "0",
	}

	// Container path (Path field)
	entry.MountPoint = d.Path

	// Host path - use HostPath if specified, otherwise assume same as container path
	entry.Device = d.HostPath
	if entry.Device == "" {
		entry.Device = d.Path
	}

	// Build mount options for device nodes
	var opts []string
	opts = append(opts, "bind")

	// Add permissions if specified (typically "rw" or "ro")
	if d.Permissions != "" {
		// CDI permissions are like "rwm" (read, write, mknod)
		// Convert to mount options
		if strings.Contains(d.Permissions, "r") && strings.Contains(d.Permissions, "w") {
			opts = append(opts, "rw")
		} else if strings.Contains(d.Permissions, "r") {
			opts = append(opts, "ro")
		}
	}

	entry.Options = strings.Join(opts, ",")
	if entry.Options == "" {
		entry.Options = "bind"
	}

	return entry
}

// ContainerEdits is a wrapper around specs.ContainerEdits for fstab conversion.
type ContainerEdits struct {
	*cdi.ContainerEdits
}

// toFstab converts all mounts and device nodes in a ContainerEdits to fstab entries.
func (e *ContainerEdits) toFstab() []FstabEntry {
	if e == nil || e.ContainerEdits == nil {
		return nil
	}

	var entries []FstabEntry

	// Convert device nodes to fstab entries
	for _, deviceNode := range e.DeviceNodes {
		if deviceNode.Path == "" {
			continue
		}
		dn := DeviceNode{deviceNode}
		entries = append(entries, dn.toFstab())
	}

	// Convert mounts to fstab entries
	for _, mount := range e.Mounts {
		if mount.ContainerPath == "" {
			continue
		}
		m := Mount{mount}
		entries = append(entries, m.toFstab())
	}

	return entries
}
