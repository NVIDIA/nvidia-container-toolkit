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

	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

type nvcdilib struct {
	logger        *logrus.Logger
	nvmllib       nvml.Interface
	devicelib     device.Interface
	deviceNamer   DeviceNamer
	driverRoot    string
	nvidiaCTKPath string
}

// New creates a new nvcdi library
func New(opts ...Option) Interface {
	l := &nvcdilib{}
	for _, opt := range opts {
		opt(l)
	}

	if l.nvmllib == nil {
		l.nvmllib = nvml.New()
	}
	if l.devicelib == nil {
		l.devicelib = device.New(device.WithNvml(l.nvmllib))
	}
	if l.logger == nil {
		l.logger = logrus.StandardLogger()
	}
	if l.deviceNamer == nil {
		l.deviceNamer, _ = NewDeviceNamer(DeviceNameStrategyIndex)
	}
	if l.driverRoot == "" {
		l.driverRoot = "/"
	}
	if l.nvidiaCTKPath == "" {
		l.nvidiaCTKPath = "/usr/bin/nvidia-ctk"
	}

	return l
}

// GetAllDeviceSpecs returns the device specs for all available devices.
func (l *nvcdilib) GetAllDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device

	gpuDeviceSpecs, err := l.getGPUDeviceSpecs()
	if err != nil {
		return nil, err
	}
	deviceSpecs = append(deviceSpecs, gpuDeviceSpecs...)

	migDeviceSpecs, err := l.getMigDeviceSpecs()
	if err != nil {
		return nil, err
	}
	deviceSpecs = append(deviceSpecs, migDeviceSpecs...)

	return deviceSpecs, nil
}

func (l *nvcdilib) getGPUDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device
	err := l.devicelib.VisitDevices(func(i int, d device.Device) error {
		deviceSpec, err := l.GetGPUDeviceSpecs(i, d)
		if err != nil {
			return err
		}
		deviceSpecs = append(deviceSpecs, *deviceSpec)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI edits for GPU devices: %v", err)
	}
	return deviceSpecs, err
}

func (l *nvcdilib) getMigDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device
	err := l.devicelib.VisitMigDevices(func(i int, d device.Device, j int, mig device.MigDevice) error {
		deviceSpec, err := l.GetMIGDeviceSpecs(i, d, j, mig)
		if err != nil {
			return err
		}
		deviceSpecs = append(deviceSpecs, *deviceSpec)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI edits for GPU devices: %v", err)
	}
	return deviceSpecs, err
}
