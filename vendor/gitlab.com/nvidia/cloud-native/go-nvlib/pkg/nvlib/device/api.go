/*
 * Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package device

import (
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

// Interface provides the API to the 'device' package
type Interface interface {
	GetDevices() ([]Device, error)
	GetMigDevices() ([]MigDevice, error)
	GetMigProfiles() ([]MigProfile, error)
	NewDevice(d nvml.Device) (Device, error)
	NewMigDevice(d nvml.Device) (MigDevice, error)
	NewMigProfile(giProfileID, ciProfileID, ciEngProfileID int, migMemorySizeMB, deviceMemorySizeBytes uint64) (MigProfile, error)
	ParseMigProfile(profile string) (MigProfile, error)
	VisitDevices(func(i int, d Device) error) error
	VisitMigDevices(func(i int, d Device, j int, m MigDevice) error) error
	VisitMigProfiles(func(p MigProfile) error) error
}

type devicelib struct {
	nvml nvml.Interface
}

var _ Interface = &devicelib{}

// New creates a new instance of the 'device' interface
func New(opts ...Option) Interface {
	d := &devicelib{}
	for _, opt := range opts {
		opt(d)
	}
	if d.nvml == nil {
		d.nvml = nvml.New()
	}
	return d
}

// WithNvml provides an Option to set the NVML library used by the 'device' interface
func WithNvml(nvml nvml.Interface) Option {
	return func(d *devicelib) {
		d.nvml = nvml
	}
}

// Option defines a function for passing options to the New() call
type Option func(*devicelib)
