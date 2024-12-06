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
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type wrapper struct {
	Interface

	vendor string
	class  string

	mergedDeviceOptions []transform.MergedDeviceOption
}

// GetSpec combines the device specs and common edits from the wrapped Interface to a single spec.Interface.
func (l *wrapper) GetSpec() (spec.Interface, error) {
	deviceSpecs, err := l.GetAllDeviceSpecs()
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

// GetAllDeviceSpecs returns the device specs for all available devices.
func (l *wrapper) GetAllDeviceSpecs() ([]specs.Device, error) {
	return l.Interface.GetAllDeviceSpecs()
}

// GetCommonEdits returns the wrapped edits and adds additional edits on top.
func (m *wrapper) GetCommonEdits() (*cdi.ContainerEdits, error) {
	edits, err := m.Interface.GetCommonEdits()
	if err != nil {
		return nil, err
	}
	edits.Env = append(edits.Env, image.EnvVarNvidiaVisibleDevices+"=void")

	return edits, nil
}
