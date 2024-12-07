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

package nvcdi

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

type imexlib nvcdilib

var _ Interface = (*imexlib)(nil)

const (
	classImexChannel = "imex-channel"
)

// GetSpec should not be called for imexlib.
func (l *imexlib) GetSpec() (spec.Interface, error) {
	return nil, fmt.Errorf("unexpected call to imexlib.GetSpec()")
}

// GetAllDeviceSpecs returns the device specs for all available devices.
func (l *imexlib) GetAllDeviceSpecs() ([]specs.Device, error) {
	channelsDiscoverer := discover.NewCharDeviceDiscoverer(
		l.logger,
		l.devRoot,
		[]string{"/dev/nvidia-caps-imex-channels/channel*"},
	)

	channels, err := channelsDiscoverer.Devices()
	if err != nil {
		return nil, err
	}

	var channelIDs []string
	for _, channel := range channels {
		channelIDs = append(channelIDs, filepath.Base(channel.Path))
	}

	return l.GetDeviceSpecsByID(channelIDs...)
}

// GetCommonEdits returns an empty set of edits for IMEX devices.
func (l *imexlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return edits.FromDiscoverer(discover.None{})
}

// GetDeviceSpecsByID returns the CDI device specs for the IMEX channels specified.
func (l *imexlib) GetDeviceSpecsByID(ids ...string) ([]specs.Device, error) {
	var deviceSpecs []specs.Device
	for _, id := range ids {
		trimmed := strings.TrimPrefix(id, "channel")
		_, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid channel ID %v: %w", id, err)
		}
		path := "/dev/nvidia-caps-imex-channels/channel" + trimmed
		deviceSpec := specs.Device{
			Name: trimmed,
			ContainerEdits: specs.ContainerEdits{
				DeviceNodes: []*specs.DeviceNode{
					{
						Path:     path,
						HostPath: filepath.Join(l.devRoot, path),
					},
				},
			},
		}
		deviceSpecs = append(deviceSpecs, deviceSpec)
	}
	return deviceSpecs, nil
}

// GetGPUDeviceEdits is unsupported for the imexlib specs
func (l *imexlib) GetGPUDeviceEdits(device.Device) (*cdi.ContainerEdits, error) {
	return nil, fmt.Errorf("GetGPUDeviceEdits is not supported")
}

// GetGPUDeviceSpecs is unsupported for the imexlib specs
func (l *imexlib) GetGPUDeviceSpecs(int, device.Device) ([]specs.Device, error) {
	return nil, fmt.Errorf("GetGPUDeviceSpecs is not supported")
}

// GetMIGDeviceEdits is unsupported for the imexlib specs
func (l *imexlib) GetMIGDeviceEdits(device.Device, device.MigDevice) (*cdi.ContainerEdits, error) {
	return nil, fmt.Errorf("GetMIGDeviceEdits is not supported")
}

// GetMIGDeviceSpecs is unsupported for the imexlib specs
func (l *imexlib) GetMIGDeviceSpecs(int, device.Device, int, device.MigDevice) ([]specs.Device, error) {
	return nil, fmt.Errorf("GetMIGDeviceSpecs is not supported")
}
