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

package edits

import (
	"fmt"

	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// New creates an Interface from the supplied options.
func New(opts ...Option) Interface {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = logger.New()
	}

	return o
}

// EditsFromDiscoverer creates CDI container edits for the specified discoverer.
func (o *options) EditsFromDiscoverer(d discover.Discover) (*cdi.ContainerEdits, error) {
	devices, err := d.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to discover devices: %w", err)
	}

	mounts, err := d.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover mounts: %w", err)
	}

	hooks, err := d.Hooks()
	if err != nil {
		return nil, fmt.Errorf("failed to discover hooks: %w", err)
	}

	c := NewContainerEdits()
	for _, d := range devices {
		edits, err := device(d).toEdits(o.allowAdditionalGIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to create container edits for device: %w", err)
		}
		c.Append(edits)
	}

	for _, m := range mounts {
		c.Append(mount(m).toEdits())
	}

	for _, h := range hooks {
		c.Append(hook(h).toEdits())
	}

	return c, nil
}

// SpecModifierFromDiscoverer creates a SpecModifier that defines the required OCI spec edits (as CDI ContainerEdits) from the specified
// discoverer.
func (o *options) SpecModifierFromDiscoverer(d discover.Discover) (oci.SpecModifier, error) {
	c, err := o.EditsFromDiscoverer(d)
	if err != nil {
		return nil, fmt.Errorf("error constructing container edits: %w", err)
	}
	e := edits{
		ContainerEdits: *c,
		logger:         o.logger,
	}

	return &e, nil
}
