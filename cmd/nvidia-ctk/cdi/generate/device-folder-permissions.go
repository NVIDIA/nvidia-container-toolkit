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

package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/sirupsen/logrus"
)

type deviceFolderPermissions struct {
	logger        *logrus.Logger
	root          string
	foldersByMode map[string][]string
}

var _ discover.Discover = (*deviceFolderPermissions)(nil)

// NewDeviceFolderPermissionHookDiscoverer creates a discoverer that can be used to update the permissions for the parent folders of nested device nodes from the specified set of device specs.
// This works around an issue with rootless podman when using crun as a low-level runtime.
// See https://github.com/containers/crun/issues/1047
// TODO: This currently assumes `root == ""`
func NewDeviceFolderPermissionHookDiscoverer(logger *logrus.Logger, root string, deviceSpecs []specs.Device) (discover.Discover, error) {
	var paths []string
	seen := make(map[string]bool)

	for _, device := range deviceSpecs {
		for _, dn := range device.ContainerEdits.DeviceNodes {
			if !strings.HasPrefix(dn.Path, "/dev") {
				logger.Warningf("Skipping unexpected device folder path for device %v", dn)
				continue
			}
			for df := filepath.Dir(dn.Path); df != "/dev"; df = filepath.Dir(df) {
				if seen[df] {
					continue
				}
				paths = append(paths, df)
				seen[df] = true
			}
		}
	}

	foldersByMode := make(map[string][]string)
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("failed to get info for path %v: %v", p, err)
		}
		mode := fmt.Sprintf("%o", info.Mode().Perm())
		foldersByMode[mode] = append(foldersByMode[mode], p)
	}

	d := &deviceFolderPermissions{
		logger:        logger,
		root:          root,
		foldersByMode: foldersByMode,
	}

	return d, nil
}

// Devices are empty for this discoverer
func (d *deviceFolderPermissions) Devices() ([]discover.Device, error) {
	return nil, nil
}

// Hooks returns a set of hooks that sets the file modes of parent folders for device nodes.
// One hook is returned per mode.
func (d *deviceFolderPermissions) Hooks() ([]discover.Hook, error) {
	locator := lookup.NewExecutableLocator(d.logger, d.root)

	var hooks []discover.Hook
	for mode, folders := range d.foldersByMode {
		args := []string{"--mode", mode}
		for _, folder := range folders {
			args = append(args, "--path", folder)
		}

		hook := discover.CreateNvidiaCTKHook(
			d.logger,
			locator,
			nvidiaCTKExecutable,
			nvidiaCTKDefaultFilePath,
			"chmod",
			args...,
		)

		hooks = append(hooks, hook)
	}

	return hooks, nil
}

// Mounts are empty for this discoverer
func (d *deviceFolderPermissions) Mounts() ([]discover.Mount, error) {
	return nil, nil
}
