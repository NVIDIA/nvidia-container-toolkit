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
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

// Interface defines the API for the nvcdi package
type Interface interface {
	GetSpec() (spec.Interface, error)
	GetCommonEdits() (*cdi.ContainerEdits, error)
	GetAllDeviceSpecs() ([]specs.Device, error)
	GetGPUDeviceEdits(device.Device) (*cdi.ContainerEdits, error)
	GetGPUDeviceSpecs(int, device.Device) ([]specs.Device, error)
	GetMIGDeviceEdits(device.Device, device.MigDevice) (*cdi.ContainerEdits, error)
	GetMIGDeviceSpecs(int, device.Device, int, device.MigDevice) ([]specs.Device, error)
	GetDeviceSpecsByID(...string) ([]specs.Device, error)
}

// A HookName refers to one of the predefined set of CDI hooks that may be
// included in the generated CDI specification.
type HookName string

const (
	// HookEnableCudaCompat refers to the hook used to enable CUDA Forward Compatibility.
	// This was added with v1.17.5 of the NVIDIA Container Toolkit.
	HookEnableCudaCompat = HookName("enable-cuda-compat")
)
