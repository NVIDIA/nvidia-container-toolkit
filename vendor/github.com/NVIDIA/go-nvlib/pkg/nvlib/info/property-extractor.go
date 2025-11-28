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

package info

import (
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
)

type propertyExtractor struct {
	root      root
	nvmllib   nvml.Interface
	devicelib device.Interface
}

var _ PropertyExtractor = &propertyExtractor{}

// HasDXCore returns true if DXCore is detected on the system.
func (i *propertyExtractor) HasDXCore() (bool, string) {
	const (
		libraryName = "libdxcore.so"
	)
	if err := i.root.assertHasLibrary(libraryName); err != nil {
		return false, fmt.Sprintf("could not load DXCore library: %v", err)
	}

	return true, "found DXCore library"
}

// HasNvml returns true if NVML is detected on the system.
func (i *propertyExtractor) HasNvml() (bool, string) {
	const (
		libraryName = "libnvidia-ml.so.1"
	)
	if err := i.root.assertHasLibrary(libraryName); err != nil {
		return false, fmt.Sprintf("could not load NVML library: %v", err)
	}

	return true, "found NVML library"
}

// IsTegraSystem returns true if the system is detected as a Tegra-based system.
//
// Deprecated: Use HasTegraFiles instead.
func (i *propertyExtractor) IsTegraSystem() (bool, string) {
	return i.HasTegraFiles()
}

// HasTegraFiles returns true if tegra-based files are detected on the system.
func (i *propertyExtractor) HasTegraFiles() (bool, string) {
	tegraReleaseFile := i.root.join("/etc/nv_tegra_release")
	tegraFamilyFile := i.root.join("/sys/devices/soc0/family")

	if info, err := os.Stat(tegraReleaseFile); err == nil && !info.IsDir() {
		return true, fmt.Sprintf("%v found", tegraReleaseFile)
	}

	if info, err := os.Stat(tegraFamilyFile); err != nil || info.IsDir() {
		return false, fmt.Sprintf("%v file not found", tegraFamilyFile)
	}

	contents, err := os.ReadFile(tegraFamilyFile)
	if err != nil {
		return false, fmt.Sprintf("could not read %v", tegraFamilyFile)
	}

	if strings.HasPrefix(strings.ToLower(string(contents)), "tegra") {
		return true, fmt.Sprintf("%v has 'tegra' prefix", tegraFamilyFile)
	}

	return false, fmt.Sprintf("%v has no 'tegra' prefix", tegraFamilyFile)
}

// HasAnIntegratedGPU checks whether any of the GPUs reported by NVML is an
// integrated GPU.
//
// As of Orin-based systems iGPUs also support limited NVML queries.
// In the absence of a robust API, we rely on heuristics based on the device
// name to make this decision.
//
// Devices with the following names are considered integrated GPUs:
//
//	GPU 0: Orin (nvgpu) (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//	GPU 0: NVIDIA Thor  (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//
// (Where this shows the nvidia-smi -L output on these systems).
func (i *propertyExtractor) HasAnIntegratedGPU() (uses bool, reason string) {
	// We ensure that this function never panics
	defer func() {
		if err := recover(); err != nil {
			uses = false
			reason = fmt.Sprintf("panic: %v", err)
		}
	}()

	ret := i.nvmllib.Init()
	if ret != nvml.SUCCESS {
		return false, fmt.Sprintf("failed to initialize nvml: %v", ret)
	}
	defer func() {
		_ = i.nvmllib.Shutdown()
	}()

	var names []string

	err := i.devicelib.VisitDevices(func(i int, d device.Device) error {
		name, ret := d.GetName()
		if ret != nvml.SUCCESS {
			return fmt.Errorf("device %v: %v", i, ret)
		}
		names = append(names, name)
		return nil
	})
	if err != nil {
		return false, fmt.Sprintf("failed to get device names: %v", err)
	}

	if len(names) == 0 {
		return false, "no devices found"
	}

	for _, name := range names {
		if IsIntegratedGPUName(name) {
			return true, fmt.Sprintf("device %q is an integrated GPU", name)
		}
	}
	return false, "no integrated GPUs found"
}

// IsIntegratedGPUName checks whether the specified device name is associated
// with a known integrated GPU.
//
// Devices with the following names are considered integrated GPUs:
//
//	GPU 0: Orin (nvgpu) (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//	GPU 0: NVIDIA Thor  (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//
// (Where this shows the nvidia-smi -L output on these systems).
func IsIntegratedGPUName(name string) bool {
	if strings.Contains(name, "(nvgpu)") {
		return true
	}
	if strings.Contains(name, "NVIDIA Thor") {
		return true
	}
	return false
}
