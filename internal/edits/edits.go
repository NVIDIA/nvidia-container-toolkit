/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package edits

import (
	"fmt"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

const (
	// An EmptyFactory is an edits factory that always returns empty CDI
	// container edits.
	EmptyFactory = empty("empty")
)

type Factory interface {
	New() *cdi.ContainerEdits
	FromDiscoverer(discover.Discover) (*cdi.ContainerEdits, error)
}

type empty string

type factory struct {
	logger                         logger.Interface
	noAdditionalGIDsForDeviceNodes bool
}

var _ Factory = (*empty)(nil)

type Option func(*factory)

func NewFactory(opts ...Option) Factory {
	f := &factory{
		logger: &logger.NullLogger{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *factory) New() *cdi.ContainerEdits {
	return EmptyFactory.New()
}

func (f *factory) FromDiscoverer(d discover.Discover) (*cdi.ContainerEdits, error) {
	devices, err := d.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to discover devices: %v", err)
	}

	envs, err := d.EnvVars()
	if err != nil {
		return nil, fmt.Errorf("failed to discover environment variables: %w", err)
	}

	mounts, err := d.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover mounts: %v", err)
	}

	hooks, err := d.Hooks()
	if err != nil {
		return nil, fmt.Errorf("failed to discover hooks: %v", err)
	}

	c := EmptyFactory.New()
	for _, d := range devices {
		edits, err := f.device(d).toEdits()
		if err != nil {
			return nil, fmt.Errorf("failed to created container edits for device: %v", err)
		}
		c.Append(edits)
	}

	for _, e := range envs {
		c.Append(envvar(e).toEdits())
	}

	for _, m := range mounts {
		c.Append(mount(m).toEdits())
	}

	for _, h := range hooks {
		c.Append(hook(h).toEdits())
	}

	return c, nil
}

func (f *factory) device(d discover.Device) *device {
	return &device{
		Device:           d,
		noAdditionalGIDs: f.noAdditionalGIDsForDeviceNodes,
	}
}

// New creates a set of empty CDI container edits for an empty factory.
func (e empty) New() *cdi.ContainerEdits {
	c := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{},
	}
	return &c
}

// FromDiscoverer creates a set of empty CDI container edits for ANY discoverer.
func (e empty) FromDiscoverer(_ discover.Discover) (*cdi.ContainerEdits, error) {
	return e.New(), nil
}

func WithLogger(logger logger.Interface) Option {
	return func(f *factory) {
		f.logger = logger
	}
}

func WithNoAdditionalGIDsForDeviceNodes(noAdditionalGIDsForDeviceNodes bool) Option {
	return func(f *factory) {
		f.noAdditionalGIDsForDeviceNodes = noAdditionalGIDsForDeviceNodes
	}
}
