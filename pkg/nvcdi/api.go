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
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

// Interface defines the API for the nvcdi package
type Interface interface {
	SpecGenerator
	GetCommonEdits() (*cdi.ContainerEdits, error)
	GetDeviceSpecsByID(...string) ([]specs.Device, error)
	// Deprecated: GetAllDeviceSpecs is deprecated. Use GetDeviceSpecsByID("all") instead.
	GetAllDeviceSpecs() ([]specs.Device, error)
}

// A SpecGenerator is used to generate a complete CDI spec for a collected set
// of devices.
type SpecGenerator interface {
	GetSpec(...string) (spec.Interface, error)
}

// A DeviceSpecGenerator is used to generate the specs for one or more devices.
type DeviceSpecGenerator interface {
	GetDeviceSpecs() ([]specs.Device, error)
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
	// FeatureEnableExplicitDriverLibraries enables the inclusion of a list of
	// explicit driver libraries.
	FeatureEnableExplicitDriverLibraries = FeatureFlag("enable-explicit-driver-libraries")
	// FeatureDisableNvsandboxUtils disables the use of nvsandboxutils when
	// querying devices.
	FeatureDisableNvsandboxUtils = FeatureFlag("disable-nvsandbox-utils")
	// FeatureEnableCoherentAnnotations enables the addition of annotations
	// coherent or non-coherent devices.
	FeatureEnableCoherentAnnotations = FeatureFlag("enable-coherent-annotations")
)
