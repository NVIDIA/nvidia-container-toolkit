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
	"path/filepath"
	"strings"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/dl"
)

type infolib struct {
	logger basicLogger
	Properties
	Resolver
}

type info struct {
	root      string
	nvmllib   nvml.Interface
	devicelib device.Interface
}

var _ Properties = &info{}
var _ Interface = &infolib{}

// HasDXCore returns true if DXCore is detected on the system.
func (i *info) HasDXCore() (bool, string) {
	const (
		libraryName = "libdxcore.so"
	)
	if err := assertHasLibrary(libraryName); err != nil {
		return false, fmt.Sprintf("could not load DXCore library: %v", err)
	}

	return true, "found DXCore library"
}

// HasNvml returns true if NVML is detected on the system
func (i *info) HasNvml() (bool, string) {
	const (
		libraryName = "libnvidia-ml.so.1"
	)
	if err := assertHasLibrary(libraryName); err != nil {
		return false, fmt.Sprintf("could not load NVML library: %v", err)
	}

	return true, "found NVML library"
}

// IsTegraSystem returns true if the system is detected as a Tegra-based system
func (i *info) IsTegraSystem() (bool, string) {
	tegraReleaseFile := filepath.Join(i.root, "/etc/nv_tegra_release")
	tegraFamilyFile := filepath.Join(i.root, "/sys/devices/soc0/family")

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

// UsesNVGPUModule checks whether the nvgpu module is used.
// This kernel module is used on Tegra-based systems when using the iGPU.
// Since some of these systems also support NVML, we use the device name
// reported by NVML to determine whether the system is an iGPU system.
//
// Devices that use the nvgpu module have their device names as:
//
//	GPU 0: Orin (nvgpu) (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//
// This function returns true if ALL devices use the nvgpu module.
func (i *info) UsesNVGPUModule() (uses bool, reason string) {
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
		if !strings.Contains(name, "(nvgpu)") {
			return false, fmt.Sprintf("device %q does not use nvgpu module", name)
		}
	}
	return true, "all devices use nvgpu module"
}

// assertHasLibrary returns an error if the specified library cannot be loaded
func assertHasLibrary(libraryName string) error {
	const (
		libraryLoadFlags = dl.RTLD_LAZY
	)
	lib := dl.New(libraryName, libraryLoadFlags)
	if err := lib.Open(); err != nil {
		return err
	}
	defer lib.Close()

	return nil
}
