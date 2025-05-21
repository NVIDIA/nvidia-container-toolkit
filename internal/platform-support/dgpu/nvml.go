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

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info/drm"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
)

type requiredInfo interface {
	GetMinorNumber() (int, error)
	GetPCIBusID() (string, error)
	getDevNodePath() (string, error)
}

func (o *options) newNvmlDGPUDiscoverer(d requiredInfo) (discover.Discover, error) {
	path, err := d.getDevNodePath()
	if err != nil {
		return nil, fmt.Errorf("error getting device node path: %w", err)
	}

	pciBusID, err := d.GetPCIBusID()
	if err != nil {
		return nil, fmt.Errorf("error getting PCI info for device: %w", err)
	}

	drmDeviceNodes, err := drm.GetDeviceNodesByBusID(pciBusID)
	if err != nil {
		return nil, fmt.Errorf("failed to determine DRM devices for %v: %v", pciBusID, err)
	}

	deviceNodePaths := append([]string{path}, drmDeviceNodes...)

	deviceNodes := discover.NewCharDeviceDiscoverer(
		o.logger,
		o.devRoot,
		deviceNodePaths,
	)

	byPathHooks := &byPathHookDiscoverer{
		logger:      o.logger,
		devRoot:     o.devRoot,
		hookCreator: o.hookCreator,
		pciBusID:    pciBusID,
		deviceNodes: deviceNodes,
	}

	dd := discover.Merge(
		deviceNodes,
		byPathHooks,
	)
	return dd, nil
}

type requiredMigInfo interface {
	getPlacementInfo() (int, int, int, error)
	getDevNodePath() (string, error)
}

func (o *options) newNvmlMigDiscoverer(d requiredMigInfo) (discover.Discover, error) {
	if o.migCaps == nil || o.migCapsError != nil {
		return nil, fmt.Errorf("error getting MIG capability device paths: %v", o.migCapsError)
	}

	gpu, gi, ci, err := d.getPlacementInfo()
	if err != nil {
		return nil, fmt.Errorf("error getting placement info: %w", err)
	}

	giCap := nvcaps.NewGPUInstanceCap(gpu, gi)
	giCapDevicePath, err := o.migCaps.GetCapDevicePath(giCap)
	if err != nil {
		return nil, fmt.Errorf("failed to get GI cap device path: %v", err)
	}

	ciCap := nvcaps.NewComputeInstanceCap(gpu, gi, ci)
	ciCapDevicePath, err := o.migCaps.GetCapDevicePath(ciCap)
	if err != nil {
		return nil, fmt.Errorf("failed to get CI cap device path: %v", err)
	}

	parentPath, err := d.getDevNodePath()
	if err != nil {
		return nil, err
	}

	deviceNodes := discover.NewCharDeviceDiscoverer(
		o.logger,
		o.devRoot,
		[]string{
			parentPath,
			giCapDevicePath,
			ciCapDevicePath,
		},
	)

	return deviceNodes, nil
}

type toRequiredInfo struct {
	device.Device
}

func (d *toRequiredInfo) GetMinorNumber() (int, error) {
	minor, ret := d.Device.GetMinorNumber()
	if ret != nvml.SUCCESS {
		return 0, ret
	}
	return minor, nil
}

func (d *toRequiredInfo) getDevNodePath() (string, error) {
	minor, err := d.GetMinorNumber()
	if err != nil {
		return "", fmt.Errorf("error getting GPU device minor number: %w", err)
	}
	path := fmt.Sprintf("/dev/nvidia%d", minor)
	return path, nil
}

type toRequiredMigInfo struct {
	device.MigDevice
	parent requiredInfo
}

func (d *toRequiredMigInfo) getPlacementInfo() (int, int, int, error) {
	gpu, err := d.parent.GetMinorNumber()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error getting GPU minor: %w", err)
	}

	gi, ret := d.GetGpuInstanceId()
	if ret != nvml.SUCCESS {
		return 0, 0, 0, fmt.Errorf("error getting GPU Instance ID: %v", ret)
	}

	ci, ret := d.GetComputeInstanceId()
	if ret != nvml.SUCCESS {
		return 0, 0, 0, fmt.Errorf("error getting Compute Instance ID: %v", ret)
	}

	return gpu, gi, ci, nil
}

func (d *toRequiredMigInfo) getDevNodePath() (string, error) {
	return d.parent.getDevNodePath()
}
