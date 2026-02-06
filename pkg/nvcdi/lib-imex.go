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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type imexlib nvcdilib

type imexChannel struct {
	id      string
	devRoot string
}

var _ deviceSpecGeneratorFactory = (*imexlib)(nil)

const (
	classImexChannel = "imex-channel"
)

// GetCommonEdits returns an empty set of edits for IMEX devices.
func (l *imexlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return l.editsFactory.FromDiscoverer(discover.None{})
}

// DeviceSpecGenerators returns the CDI device spec generators for the specified
// imex channel IDs.
// Valid IDs are:
// * numeric channel IDs
// * channel<numericChannelID>
// * the special ID 'all'
func (l *imexlib) DeviceSpecGenerators(ids ...string) (DeviceSpecGenerator, error) {
	channelsIDs, err := l.getChannelIDs(ids...)
	if err != nil {
		return nil, err
	}

	var deviceSpecGenerators DeviceSpecGenerators
	for _, id := range channelsIDs {
		deviceSpecGenerators = append(deviceSpecGenerators, &imexChannel{id: id, devRoot: l.devRoot})
	}

	return deviceSpecGenerators, nil
}

func (l *imexlib) getChannelIDs(ids ...string) ([]string, error) {
	var channelIDs []string
	for _, id := range ids {
		trimmed := strings.TrimPrefix(id, "channel")
		if trimmed == "all" {
			return l.getAllChannelIDs()
		}
		_, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid channel ID %v: %w", id, err)
		}
		channelIDs = append(channelIDs, trimmed)
	}
	return channelIDs, nil
}

// getAllChannelIDs returns the device IDs for all available IMEX channels.
func (l *imexlib) getAllChannelIDs() ([]string, error) {
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
		channelID := filepath.Base(channel.Path)
		channelIDs = append(channelIDs, strings.TrimPrefix(channelID, "channel"))
	}

	return channelIDs, nil
}

// GetDeviceSpecs returns the CDI device specs the specified IMEX channel.
func (l *imexChannel) GetDeviceSpecs() ([]specs.Device, error) {
	path := "/dev/nvidia-caps-imex-channels/channel" + l.id
	deviceSpec := specs.Device{
		Name: l.id,
		ContainerEdits: specs.ContainerEdits{
			DeviceNodes: []*specs.DeviceNode{
				{
					Path:     path,
					HostPath: filepath.Join(l.devRoot, path),
				},
			},
		},
	}
	return []specs.Device{deviceSpec}, nil
}
