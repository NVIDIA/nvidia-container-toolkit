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
)

type requiredInfo interface {
	GetMinorNumber() (int, error)
	GetPCIBusID() (string, error)
}

func (o *options) newNvmlDGPUDiscoverer(d requiredInfo) (discover.Discover, error) {
	minor, err := d.GetMinorNumber()
	if err != nil {
		return nil, fmt.Errorf("error getting GPU device minor number: %w", err)
	}
	path := fmt.Sprintf("/dev/nvidia%d", minor)

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
		logger:            o.logger,
		devRoot:           o.devRoot,
		nvidiaCDIHookPath: o.nvidiaCDIHookPath,
		pciBusID:          pciBusID,
		deviceNodes:       deviceNodes,
	}

	dd := discover.Merge(
		deviceNodes,
		byPathHooks,
	)
	return dd, nil
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
