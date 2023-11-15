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

package info

import (
	"fmt"
	"strings"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
)

// additionalInfo allows for the info.Interface to be extened to implement the infoInterface.
type additionalInfo struct {
	info.Interface
	nvmllib   nvml.Interface
	devicelib device.Interface
}

// UsesNVGPUModule checks whether the nvgpu module is used.
// We use the device name to signal this, since devices that use the nvgpu module have their device
// names as:
//
//	GPU 0: Orin (nvgpu) (UUID: 54d0709b-558d-5a59-9c65-0c5fc14a21a4)
//
// This function returns true if ALL devices use the nvgpu module.
func (i additionalInfo) UsesNVGPUModule() (uses bool, reason string) {
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
