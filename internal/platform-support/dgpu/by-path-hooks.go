/**
# Copyright 2024 NVIDIA CORPORATION
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

package dgpu

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// byPathHookDiscoverer discovers the entities required for injecting by-path DRM device links
type byPathHookDiscoverer struct {
	logger            logger.Interface
	devRoot           string
	nvidiaCDIHookPath string
	pciBusID          string
	deviceNodes       discover.Discover
}

var _ discover.Discover = (*byPathHookDiscoverer)(nil)

// Devices returns the empty list for the by-path hook discoverer
func (d *byPathHookDiscoverer) Devices() ([]discover.Device, error) {
	return nil, nil
}

// Hooks returns the hooks for the GPU device.
// The following hooks are detected:
//  1. A hook to create /dev/dri/by-path symlinks
func (d *byPathHookDiscoverer) Hooks() ([]discover.Hook, error) {
	links, err := d.deviceNodeLinks()
	if err != nil {
		return nil, fmt.Errorf("failed to discover DRA device links: %v", err)
	}
	if len(links) == 0 {
		return nil, nil
	}

	var args []string
	for _, l := range links {
		args = append(args, "--link", l)
	}

	hook := discover.CreateNvidiaCDIHook(
		d.nvidiaCDIHookPath,
		"create-symlinks",
		args...,
	)

	return []discover.Hook{hook}, nil
}

// Mounts returns an empty slice for a full GPU
func (d *byPathHookDiscoverer) Mounts() ([]discover.Mount, error) {
	return nil, nil
}

func (d *byPathHookDiscoverer) deviceNodeLinks() ([]string, error) {
	devices, err := d.deviceNodes.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to discover device nodes: %v", err)
	}

	if len(devices) == 0 {
		return nil, nil
	}

	selectedDevices := make(map[string]bool)
	for _, d := range devices {
		selectedDevices[d.HostPath] = true
	}

	candidates := []string{
		fmt.Sprintf("/dev/dri/by-path/pci-%s-card", d.pciBusID),
		fmt.Sprintf("/dev/dri/by-path/pci-%s-render", d.pciBusID),
	}

	var links []string
	for _, c := range candidates {
		linkPath := filepath.Join(d.devRoot, c)
		device, err := os.Readlink(linkPath)
		if err != nil {
			d.logger.Warningf("Failed to evaluate symlink %v; ignoring", linkPath)
			continue
		}

		deviceNode := device
		if !filepath.IsAbs(device) {
			deviceNode = filepath.Join(filepath.Dir(linkPath), device)
		}
		if !selectedDevices[deviceNode] {
			d.logger.Debugf("ignoring device symlink %v -> %v since %v is not mounted", linkPath, device, deviceNode)
			continue
		}
		d.logger.Debugf("adding device symlink %v -> %v", linkPath, device)
		links = append(links, fmt.Sprintf("%v::%v", device, linkPath))
	}

	return links, nil
}
