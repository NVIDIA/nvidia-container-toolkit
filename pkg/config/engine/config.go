/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package engine

// A Config represents a config for a container engine.
// These include container, cri-o, and docker.
// The config is logically split into a Source and Destination. This allows an
// existing config to be updated (Source == Destination) or runtime-specific
// settings to be written to a new config. The latter is useful when creation
// NVIDIA-specific drop-in files for container engines that support this.
type Config struct {
	Source      RuntimeConfigSource
	Destination RuntimeConfigDestination
}

// A RuntimeConfigSource allows runtime-specific settings to be READ from a
// config.
type RuntimeConfigSource interface {
	DefaultRuntime() string
	GetRuntimeConfig(string) (RuntimeConfig, error)
	GetDefaultRuntimeOptions() interface{}
	String() string
}

// A RuntimeConfigDestination allows a runtime with specific settings to be
// WRITTEN to a config.
type RuntimeConfigDestination interface {
	AddRuntimeWithOptions(string, string, bool, interface{}) error
	EnableCDI()
	RemoveRuntime(string) error
	UpdateDefaultRuntime(string, string) error
	Save(string) (int64, error)
	String() string
}

// AddRuntime adds a runtime to the destination config and optionally sets it as the default.
// The options to apply to the added runtime are read from the source config
// default runtime.
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	options := c.Source.GetDefaultRuntimeOptions()
	return c.Destination.AddRuntimeWithOptions(name, path, setAsDefault, options)
}

// RemoveRuntime removes a runtime from the destination config.
func (c *Config) RemoveRuntime(runtime string) error {
	return c.Destination.RemoveRuntime(runtime)
}

// UpdateDefaultRuntime updates the default runtime setting in the destination config.
// When action is 'set' the provided runtime name is set as the default.
// When action is 'unset' we make sure the provided runtime name is not
// the default.
func (c *Config) UpdateDefaultRuntime(runtime string, action string) error {
	return c.Destination.UpdateDefaultRuntime(runtime, action)
}

// EnableCDI enables CDI in the destination config.
func (c *Config) EnableCDI() {
	c.Destination.EnableCDI()
}

// DefaultRuntime returns the default runtime for the source config.
func (c *Config) DefaultRuntime() string {
	return c.Source.DefaultRuntime()
}

// GetRuntimeConfig returns the source config for the specified runtime.
func (c *Config) GetRuntimeConfig(runtime string) (RuntimeConfig, error) {
	return c.Source.GetRuntimeConfig(runtime)
}

// Save saves the destination runtime to the specified path.
func (c *Config) Save(path string) (int64, error) {
	return c.Destination.Save(path)
}

func (c *Config) String() string {
	return c.Destination.String()
}
