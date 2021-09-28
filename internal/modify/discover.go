/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package modify

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	log "github.com/sirupsen/logrus"
)

type discoverModifier struct {
	logger    *log.Logger
	discover  discover.Discover
	root      string
	bundleDir string
}

var _ Modifier = (*discoverModifier)(nil)

// NewModifierFor creates a Modifier that can be used to apply the modifications to an OCI specification
// required by the specified Discover instance.
func NewModifierFor(discover discover.Discover, root string, bundleDir string) Modifier {
	return NewModifierWithLoggerFor(log.StandardLogger(), discover, root, bundleDir)
}

// NewModifierWithLoggerFor creates a Modifier that can be used to apply the modifications to an OCI specification
// required by the specified Discover instance.
func NewModifierWithLoggerFor(logger *log.Logger, discover discover.Discover, root string, bundleDir string) Modifier {
	m := discoverModifier{
		logger:    logger,
		discover:  discover,
		root:      root,
		bundleDir: bundleDir,
	}

	return &m
}

// Modify applies the modifications for the discovered devices, mounts, etc. to the specified
// OCI spec.
func (m discoverModifier) Modify(spec oci.Spec) error {
	m.logger.Infof("Determining required OCI spec modifications")
	modifiers, err := m.modifiers()
	if err != nil {
		return fmt.Errorf("error constructing modifiers: %v", err)
	}

	m.logger.Infof("Applying %v modifications", len(modifiers))
	for _, mi := range modifiers {
		err := mi.Modify(spec)
		if err != nil {
			return fmt.Errorf("could not apply modifier %v: %v", mi, err)
		}
	}
	return nil
}

func (m discoverModifier) modifiers() ([]Modifier, error) {
	var modifiers []Modifier

	deviceModifiers, err := m.deviceModifiers()
	if err != nil {
		return nil, err
	}
	modifiers = append(modifiers, deviceModifiers...)

	mountModifiers, err := m.mountModifiers()
	if err != nil {
		return nil, err
	}
	modifiers = append(modifiers, mountModifiers...)

	hookModifiers, err := m.hookModifiers()
	if err != nil {
		return nil, err
	}
	modifiers = append(modifiers, hookModifiers...)

	return modifiers, nil
}

func (m discoverModifier) deviceModifiers() ([]Modifier, error) {
	var modifiers []Modifier

	devices, err := m.discover.Devices()
	if err != nil {
		return nil, fmt.Errorf("error discovering devices: %v", err)
	}

	for _, d := range devices {
		m := Device{
			logger: m.logger,
			Device: d,
		}
		modifiers = append(modifiers, m)
	}

	return modifiers, nil
}

func (m discoverModifier) mountModifiers() ([]Modifier, error) {
	var modifiers []Modifier

	mounts, err := m.discover.Mounts()
	if err != nil {
		return nil, fmt.Errorf("error discovering mounts: %v", err)
	}

	for _, mi := range mounts {
		mm := Mount{
			logger: m.logger,
			Mount:  mi,
			root:   m.root,
		}
		modifiers = append(modifiers, mm)
	}

	return modifiers, nil
}

func (m discoverModifier) hookModifiers() ([]Modifier, error) {
	var modifiers []Modifier

	hooks, err := m.discover.Hooks()
	if err != nil {
		return nil, fmt.Errorf("error discovering hooks: %v", err)
	}

	for _, h := range hooks {
		m := Hook{
			logger:    m.logger,
			Hook:      h,
			bundleDir: m.bundleDir,
		}
		modifiers = append(modifiers, m)
	}

	return modifiers, nil
}
