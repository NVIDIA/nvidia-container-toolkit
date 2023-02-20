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
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

type nvcdilib struct {
	logger        *logrus.Logger
	nvmllib       nvml.Interface
	mode          string
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
	if l.mode == "" {
		l.mode = "nvml"
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

	switch l.mode {
	case "nvml":
		if l.nvmllib == nil {
			l.nvmllib = nvml.New()
		}
		if l.devicelib == nil {
			l.devicelib = device.New(device.WithNvml(l.nvmllib))
		}

		return (*nvmllib)(l)
	case "wsl":
		return (*wsllib)(l)
	}

	// TODO: We want an error here.
	return nil
}
