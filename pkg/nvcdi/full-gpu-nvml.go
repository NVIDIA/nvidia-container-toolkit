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

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/dgpu"
)

// A fullGPUDeviceSpecGenerator generates the CDI device specifications for a
// single full GPU.
type fullGPUDeviceSpecGenerator struct {
	*nvmllib
	uuid  string
	index int

	featureFlags          map[FeatureFlag]bool
	additionalDiscoverers []discover.Discover
}

var _ DeviceSpecGenerator = (*fullGPUDeviceSpecGenerator)(nil)

func (l *fullGPUDeviceSpecGenerator) GetUUID() (string, error) {
	return l.uuid, nil
}

func (l *nvmllib) newFullGPUDeviceSpecGeneratorFromDevice(index int, d device.Device, featureFlags map[FeatureFlag]bool) (*fullGPUDeviceSpecGenerator, error) {
	uuid, ret := d.GetUUID()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device UUID: %v", ret)
	}
	e := &fullGPUDeviceSpecGenerator{
		nvmllib: l,
		uuid:    uuid,
		index:   index,

		featureFlags: featureFlags,
	}

	return e, nil
}

func (l *nvmllib) newFullGPUDeviceSpecGeneratorFromNVMLDevice(uuid string, nvmlDevice nvml.Device, featureFlags map[FeatureFlag]bool) (DeviceSpecGenerator, error) {
	index, ret := nvmlDevice.GetIndex()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device index: %v", ret)
	}

	e := &fullGPUDeviceSpecGenerator{
		nvmllib: l,
		uuid:    uuid,
		index:   index,

		featureFlags: featureFlags,
	}
	return e, nil
}

func (l *fullGPUDeviceSpecGenerator) GetDeviceSpecs() ([]specs.Device, error) {
	deviceEdits, err := l.getDeviceEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to get CDI device edits: %w", err)
	}

	names, err := l.getNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get device names: %w", err)
	}

	annotations, err := l.getDeviceAnnotations()
	if err != nil {
		l.logger.Warningf("Ignoring error getting device annotations for device(s) %v: %v", names, err)
		annotations = nil
	}
	var deviceSpecs []specs.Device
	for _, name := range names {
		deviceSpec := specs.Device{
			Name:           name,
			ContainerEdits: *deviceEdits.ContainerEdits,
			Annotations:    annotations,
		}
		deviceSpecs = append(deviceSpecs, deviceSpec)
	}

	return deviceSpecs, nil
}

func (l *fullGPUDeviceSpecGenerator) device() (device.Device, error) {
	return l.devicelib.NewDeviceByUUID(l.uuid)
}

func (l *fullGPUDeviceSpecGenerator) getDeviceAnnotations() (map[string]string, error) {
	if !l.featureFlags[FeatureEnableCoherentAnnotations] {
		return nil, nil
	}

	device, err := l.device()
	if err != nil {
		return nil, err
	}

	// TODO: Should we distinguish between not-supported and disabled?
	isCoherent, err := device.IsCoherent()
	if err != nil {
		return nil, fmt.Errorf("failed to check device coherence: %w", err)
	}

	annotations := map[string]string{
		"gpu.nvidia.com/coherent": fmt.Sprintf("%v", isCoherent),
	}

	return annotations, nil
}

// GetGPUDeviceEdits returns the CDI edits for the full GPU represented by 'device'.
func (l *fullGPUDeviceSpecGenerator) getDeviceEdits() (*cdi.ContainerEdits, error) {
	device, err := l.device()
	if err != nil {
		return nil, err
	}

	deviceDiscoverer, err := l.newFullGPUDiscoverer(device)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}
	editsForDevice, err := l.editsFactory.FromDiscoverer(deviceDiscoverer)
	if err != nil {
		return nil, fmt.Errorf("failed to create container edits for device: %v", err)
	}

	return editsForDevice, nil
}

func (l *fullGPUDeviceSpecGenerator) getNames() ([]string, error) {
	return l.deviceNamers.GetDeviceNames(l.index, l)
}

// newFullGPUDiscoverer creates a discoverer for the full GPU defined by the specified device.
func (l *fullGPUDeviceSpecGenerator) newFullGPUDiscoverer(d device.Device) (discover.Discover, error) {
	deviceNodes, err := dgpu.NewForDevice(d,
		dgpu.WithDevRoot(l.devRoot),
		dgpu.WithLogger(l.logger),
		dgpu.WithHookCreator(l.hookCreator),
		dgpu.WithNvsandboxuitilsLib(l.nvsandboxutilslib),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device discoverer: %v", err)
	}

	deviceFolderPermissionHooks := newDeviceFolderPermissionHookDiscoverer(
		l.logger,
		l.devRoot,
		l.hookCreator,
		deviceNodes,
	)

	var discoverers []discover.Discover

	discoverers = append(discoverers,
		deviceNodes,
		deviceFolderPermissionHooks,
	)

	discoverers = append(discoverers, l.additionalDiscoverers...)

	dd := discover.Merge(
		discoverers...,
	)

	return dd, nil
}
