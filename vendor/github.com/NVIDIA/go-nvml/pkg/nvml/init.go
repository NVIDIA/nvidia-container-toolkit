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
	if lib == nil {
		panic(fmt.Sprintf("error instantiating DynamicLibrary for %s", nvmlLibraryName))
	}

	err := lib.Open()
	if err != nil {
		panic(fmt.Sprintf("error opening %s: %v", nvmlLibraryName, err))
	}

	nvml = lib
	updateVersionedSymbols()

	return nvmlInit()
}

// nvml.InitWithFlags()
func InitWithFlags(Flags uint32) Return {
	lib := dl.New(nvmlLibraryName, nvmlLibraryLoadFlags)
	if lib == nil {
		panic(fmt.Sprintf("error instantiating DynamicLibrary for %s", nvmlLibraryName))
	}

	err := lib.Open()
	if err != nil {
		panic(fmt.Sprintf("error opening %s: %v", nvmlLibraryName, err))
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
var nvmlDeviceGetComputeRunningProcesses = nvmlDeviceGetComputeRunningProcesses_v1
var nvmlDeviceGetGraphicsRunningProcesses = nvmlDeviceGetGraphicsRunningProcesses_v1

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
	err = nvml.Lookup("nvmlEventSetWait_v2")
	if err == nil {
		nvmlEventSetWait = nvmlEventSetWait_v2
	}
	err = nvml.Lookup("nvmlDeviceGetAttributes_v2")
	if err == nil {
		nvmlDeviceGetAttributes = nvmlDeviceGetAttributes_v2
	}
	err = nvml.Lookup("nvmlDeviceGetComputeRunningProcesses_v2")
	if err == nil {
		nvmlDeviceGetComputeRunningProcesses = nvmlDeviceGetComputeRunningProcesses_v2
	}
	err = nvml.Lookup("nvmlDeviceGetGraphicsRunningProcesses_v2")
	if err == nil {
		nvmlDeviceGetGraphicsRunningProcesses = nvmlDeviceGetGraphicsRunningProcesses_v2
	}

}
