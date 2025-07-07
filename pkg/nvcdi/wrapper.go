/**
# Copyright 2025 NVIDIA CORPORATION
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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type wrapper struct {
	factory deviceSpecGeneratorFactory

	vendor string
	class  string

	mergedDeviceOptions []transform.MergedDeviceOption
}

// TODO: Rename this type
type deviceSpecGeneratorFactory interface {
	DeviceSpecGenerators(...string) (DeviceSpecGenerator, error)
	GetCommonEdits() (*cdi.ContainerEdits, error)
}

// DeviceSpecGenerators can be used to combine multiple device spec generators.
// This type also implements the DeviceSpecGenerator interface.
type DeviceSpecGenerators []DeviceSpecGenerator

var _ DeviceSpecGenerator = (DeviceSpecGenerators)(nil)

// GetSpec combines the device specs and common edits from the wrapped Interface to a single spec.Interface.
func (l *wrapper) GetSpec(devices ...string) (spec.Interface, error) {
	if len(devices) == 0 {
		devices = append(devices, "all")
	}
	deviceSpecs, err := l.GetDeviceSpecsByID(devices...)
	if err != nil {
		return nil, err
	}

	edits, err := l.GetCommonEdits()
	if err != nil {
		return nil, err
	}

	return spec.New(
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*edits.ContainerEdits),
		spec.WithVendor(l.vendor),
		spec.WithClass(l.class),
		spec.WithMergedDeviceOptions(l.mergedDeviceOptions...),
	)
}

// GetDeviceSpecsByID returns the CDI device specs for devices with the
// specified IDs.
// The device IDs are interpreted by the configured factory.
func (l *wrapper) GetDeviceSpecsByID(devices ...string) ([]specs.Device, error) {
	generators, err := l.factory.DeviceSpecGenerators(devices...)
	if err != nil {
		return nil, fmt.Errorf("failed to construct device spec generators: %w", err)
	}
	return generators.GetDeviceSpecs()
}

// GetAllDeviceSpecs returns the device specs for all available devices.
//
// Deprecated: Use GetDeviceSpecsByID("all") instead.
func (l *wrapper) GetAllDeviceSpecs() ([]specs.Device, error) {
	return l.GetDeviceSpecsByID("all")
}

// GetCommonEdits returns the wrapped edits and adds additional edits on top.
func (m *wrapper) GetCommonEdits() (*cdi.ContainerEdits, error) {
	edits, err := m.factory.GetCommonEdits()
	if err != nil {
		return nil, err
	}
	edits.Env = append(edits.Env, image.EnvVarNvidiaVisibleDevices+"=void")

	return edits, nil
}

// GetDeviceSpecs returns the combined specs for each device spec generator.
func (g DeviceSpecGenerators) GetDeviceSpecs() ([]specs.Device, error) {
	var allDeviceSpecs []specs.Device
	for _, dsg := range g {
		if dsg == nil {
			continue
		}
		deviceSpecs, err := dsg.GetDeviceSpecs()
		if err != nil {
			return nil, err
		}
		allDeviceSpecs = append(allDeviceSpecs, deviceSpecs...)
	}

	return allDeviceSpecs, nil
}
