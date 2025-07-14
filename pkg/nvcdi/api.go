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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

// Interface defines the API for the nvcdi package
type Interface interface {
	GetSpec(...string) (spec.Interface, error)
	GetCommonEdits() (*cdi.ContainerEdits, error)
	GetAllDeviceSpecs() ([]specs.Device, error)
	GetGPUDeviceEdits(device.Device) (*cdi.ContainerEdits, error)
	GetGPUDeviceSpecs(int, device.Device) ([]specs.Device, error)
	GetMIGDeviceEdits(device.Device, device.MigDevice) (*cdi.ContainerEdits, error)
	GetMIGDeviceSpecs(int, device.Device, int, device.MigDevice) ([]specs.Device, error)
	GetDeviceSpecsByID(...string) ([]specs.Device, error)
}

// A HookName represents one of the predefined NVIDIA CDI hooks.
type HookName = discover.HookName

const (
	// AllHooks is a special hook name that allows all hooks to be matched.
	AllHooks = discover.AllHooks

	// A CreateSymlinksHook is used to create symlinks in the container.
	CreateSymlinksHook = discover.CreateSymlinksHook
	// DisableDeviceNodeModificationHook refers to the hook used to ensure that
	// device nodes are not created by libnvidia-ml.so or nvidia-smi in a
	// container.
	// Added in v1.17.8
	DisableDeviceNodeModificationHook = discover.DisableDeviceNodeModificationHook
	// An EnableCudaCompatHook is used to enabled CUDA Forward Compatibility.
	// Added in v1.17.5
	EnableCudaCompatHook = discover.EnableCudaCompatHook
	// An UpdateLDCacheHook is used to update the ldcache in the container.
	UpdateLDCacheHook = discover.UpdateLDCacheHook

	// Deprecated: Use CreateSymlinksHook instead.
	HookCreateSymlinks = CreateSymlinksHook
	// Deprecated: Use EnableCudaCompatHook instead.
	HookEnableCudaCompat = EnableCudaCompatHook
	// Deprecated: Use UpdateLDCacheHook instead.
	HookUpdateLDCache = UpdateLDCacheHook
)

// A FeatureFlag refers to a specific feature that can be toggled in the CDI api.
// All features are off by default.
type FeatureFlag string

const (
	// FeatureDisableNvsandboxUtils disables the use of nvsandboxutils when
	// querying devices.
	FeatureDisableNvsandboxUtils = FeatureFlag("disable-nvsandbox-utils")
)
