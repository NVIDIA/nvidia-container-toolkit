// Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nvml

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

import "C"

const (
	nvmlLibraryName      = "libnvidia-ml.so.1"
	nvmlLibraryLoadFlags = dl.RTLD_LAZY | dl.RTLD_GLOBAL
)

var nvml *dl.DynamicLibrary

// nvml.Init()
func Init() Return {
	lib := dl.New(nvmlLibraryName, nvmlLibraryLoadFlags)
	err := lib.Open()
	if err != nil {
		return ERROR_LIBRARY_NOT_FOUND
	}

	nvml = lib
	updateVersionedSymbols()

	return nvmlInit()
}

// nvml.InitWithFlags()
func InitWithFlags(Flags uint32) Return {
	lib := dl.New(nvmlLibraryName, nvmlLibraryLoadFlags)
	err := lib.Open()
	if err != nil {
		return ERROR_LIBRARY_NOT_FOUND
	}

	nvml = lib

	return nvmlInitWithFlags(Flags)
}

// nvml.Shutdown()
func Shutdown() Return {
	ret := nvmlShutdown()
	if ret != SUCCESS {
		return ret
	}

	err := nvml.Close()
	if err != nil {
		panic(fmt.Sprintf("error closing %s: %v", nvmlLibraryName, err))
	}

	return ret
}

// Default all versioned APIs to v1 (to infer the types)
var nvmlInit = nvmlInit_v1
var nvmlDeviceGetPciInfo = nvmlDeviceGetPciInfo_v1
var nvmlDeviceGetCount = nvmlDeviceGetCount_v1
var nvmlDeviceGetHandleByIndex = nvmlDeviceGetHandleByIndex_v1
var nvmlDeviceGetHandleByPciBusId = nvmlDeviceGetHandleByPciBusId_v1
var nvmlDeviceGetNvLinkRemotePciInfo = nvmlDeviceGetNvLinkRemotePciInfo_v1
var nvmlDeviceRemoveGpu = nvmlDeviceRemoveGpu_v1
var nvmlDeviceGetGridLicensableFeatures = nvmlDeviceGetGridLicensableFeatures_v1
var nvmlEventSetWait = nvmlEventSetWait_v1
var nvmlDeviceGetAttributes = nvmlDeviceGetAttributes_v1
var nvmlComputeInstanceGetInfo = nvmlComputeInstanceGetInfo_v1
var DeviceGetComputeRunningProcesses = deviceGetComputeRunningProcesses_v1
var DeviceGetGraphicsRunningProcesses = deviceGetGraphicsRunningProcesses_v1
var DeviceGetMPSComputeRunningProcesses = deviceGetMPSComputeRunningProcesses_v1
var GetBlacklistDeviceCount = GetExcludedDeviceCount
var GetBlacklistDeviceInfoByIndex = GetExcludedDeviceInfoByIndex
var nvmlDeviceGetGpuInstancePossiblePlacements = nvmlDeviceGetGpuInstancePossiblePlacements_v1
var nvmlVgpuInstanceGetLicenseInfo = nvmlVgpuInstanceGetLicenseInfo_v1

type BlacklistDeviceInfo = ExcludedDeviceInfo
type ProcessInfo_v1Slice []ProcessInfo_v1
type ProcessInfo_v2Slice []ProcessInfo_v2

func (pis ProcessInfo_v1Slice) ToProcessInfoSlice() []ProcessInfo {
	var newInfos []ProcessInfo
	for _, pi := range pis {
		info := ProcessInfo{
			Pid:               pi.Pid,
			UsedGpuMemory:     pi.UsedGpuMemory,
			GpuInstanceId:     0xFFFFFFFF, // GPU instance ID is invalid in v1
			ComputeInstanceId: 0xFFFFFFFF, // Compute instance ID is invalid in v1
		}
		newInfos = append(newInfos, info)
	}
	return newInfos
}

func (pis ProcessInfo_v2Slice) ToProcessInfoSlice() []ProcessInfo {
	var newInfos []ProcessInfo
	for _, pi := range pis {
		info := ProcessInfo{
			Pid:               pi.Pid,
			UsedGpuMemory:     pi.UsedGpuMemory,
			GpuInstanceId:     pi.GpuInstanceId,
			ComputeInstanceId: pi.ComputeInstanceId,
		}
		newInfos = append(newInfos, info)
	}
	return newInfos
}

// updateVersionedSymbols()
func updateVersionedSymbols() {
	err := nvml.Lookup("nvmlInit_v2")
	if err == nil {
		nvmlInit = nvmlInit_v2
	}
	err = nvml.Lookup("nvmlDeviceGetPciInfo_v2")
	if err == nil {
		nvmlDeviceGetPciInfo = nvmlDeviceGetPciInfo_v2
	}
	err = nvml.Lookup("nvmlDeviceGetPciInfo_v3")
	if err == nil {
		nvmlDeviceGetPciInfo = nvmlDeviceGetPciInfo_v3
	}
	err = nvml.Lookup("nvmlDeviceGetCount_v2")
	if err == nil {
		nvmlDeviceGetCount = nvmlDeviceGetCount_v2
	}
	err = nvml.Lookup("nvmlDeviceGetHandleByIndex_v2")
	if err == nil {
		nvmlDeviceGetHandleByIndex = nvmlDeviceGetHandleByIndex_v2
	}
	err = nvml.Lookup("nvmlDeviceGetHandleByPciBusId_v2")
	if err == nil {
		nvmlDeviceGetHandleByPciBusId = nvmlDeviceGetHandleByPciBusId_v2
	}
	err = nvml.Lookup("nvmlDeviceGetNvLinkRemotePciInfo_v2")
	if err == nil {
		nvmlDeviceGetNvLinkRemotePciInfo = nvmlDeviceGetNvLinkRemotePciInfo_v2
	}
	// Unable to overwrite nvmlDeviceRemoveGpu() because the v2 function takes
	// a different set of parameters than the v1 function.
	//err = nvml.Lookup("nvmlDeviceRemoveGpu_v2")
	//if err == nil {
	//    nvmlDeviceRemoveGpu = nvmlDeviceRemoveGpu_v2
	//}
	err = nvml.Lookup("nvmlDeviceGetGridLicensableFeatures_v2")
	if err == nil {
		nvmlDeviceGetGridLicensableFeatures = nvmlDeviceGetGridLicensableFeatures_v2
	}
	err = nvml.Lookup("nvmlDeviceGetGridLicensableFeatures_v3")
	if err == nil {
		nvmlDeviceGetGridLicensableFeatures = nvmlDeviceGetGridLicensableFeatures_v3
	}
	err = nvml.Lookup("nvmlDeviceGetGridLicensableFeatures_v4")
	if err == nil {
		nvmlDeviceGetGridLicensableFeatures = nvmlDeviceGetGridLicensableFeatures_v4
	}
	err = nvml.Lookup("nvmlEventSetWait_v2")
	if err == nil {
		nvmlEventSetWait = nvmlEventSetWait_v2
	}
	err = nvml.Lookup("nvmlDeviceGetAttributes_v2")
	if err == nil {
		nvmlDeviceGetAttributes = nvmlDeviceGetAttributes_v2
	}
	err = nvml.Lookup("nvmlComputeInstanceGetInfo_v2")
	if err == nil {
		nvmlComputeInstanceGetInfo = nvmlComputeInstanceGetInfo_v2
	}
	err = nvml.Lookup("nvmlDeviceGetComputeRunningProcesses_v2")
	if err == nil {
		DeviceGetComputeRunningProcesses = deviceGetComputeRunningProcesses_v2
	}
	err = nvml.Lookup("nvmlDeviceGetComputeRunningProcesses_v3")
	if err == nil {
		DeviceGetComputeRunningProcesses = deviceGetComputeRunningProcesses_v3
	}
	err = nvml.Lookup("nvmlDeviceGetGraphicsRunningProcesses_v2")
	if err == nil {
		DeviceGetGraphicsRunningProcesses = deviceGetGraphicsRunningProcesses_v2
	}
	err = nvml.Lookup("nvmlDeviceGetGraphicsRunningProcesses_v3")
	if err == nil {
		DeviceGetGraphicsRunningProcesses = deviceGetGraphicsRunningProcesses_v3
	}
	err = nvml.Lookup("nvmlDeviceGetMPSComputeRunningProcesses_v2")
	if err == nil {
		DeviceGetMPSComputeRunningProcesses = deviceGetMPSComputeRunningProcesses_v2
	}
	err = nvml.Lookup("nvmlDeviceGetMPSComputeRunningProcesses_v3")
	if err == nil {
		DeviceGetMPSComputeRunningProcesses = deviceGetMPSComputeRunningProcesses_v3
	}
	err = nvml.Lookup("nvmlDeviceGetGpuInstancePossiblePlacements_v2")
	if err == nil {
		nvmlDeviceGetGpuInstancePossiblePlacements = nvmlDeviceGetGpuInstancePossiblePlacements_v2
	}
	err = nvml.Lookup("nvmlVgpuInstanceGetLicenseInfo_v2")
	if err == nil {
		nvmlVgpuInstanceGetLicenseInfo = nvmlVgpuInstanceGetLicenseInfo_v2
	}
}
