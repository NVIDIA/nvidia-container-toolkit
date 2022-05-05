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

package cuda

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

/*
#cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files

#ifdef _WIN32
#define CUDAAPI __stdcall
#else
#define CUDAAPI
#endif

typedef int CUdevice;

typedef enum CUdevice_attribute_enum {
    CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MAJOR = 75,
    CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MINOR = 76
} CUdevice_attribute;

typedef enum cudaError_enum {
	CUDA_SUCCESS = 0
} CUresult;

CUresult CUDAAPI cuInit(unsigned int Flags);
CUresult CUDAAPI cuDriverGetVersion(int *driverVersion);
CUresult CUDAAPI cuDeviceGet(CUdevice *device, int ordinal);
CUresult CUDAAPI cuDeviceGetAttribute(int *pi, CUdevice_attribute attrib, CUdevice dev);
*/
import "C"

const (
	libraryName      = "libcuda.so.1"
	libraryLoadFlags = dl.RTLD_LAZY | dl.RTLD_GLOBAL
)

// cuda stores a reference the cuda dynamic library
var lib *dl.DynamicLibrary

// Version returns the CUDA version of the driver as a string or an error if this
// cannot be determined.
func Version() (string, error) {
	lib, err := load()
	if err != nil {
		return "", err
	}
	defer lib.Close()

	if err := lib.Lookup("cuDriverGetVersion"); err != nil {
		return "", fmt.Errorf("failed to lookup symbol: %v", err)
	}

	var version C.int
	if result := C.cuDriverGetVersion(&version); result != C.CUDA_SUCCESS {
		return "", fmt.Errorf("failed to get CUDA version: result=%v", result)
	}

	major := version / 1000
	minor := version % 100 / 10

	return fmt.Sprintf("%d.%d", major, minor), nil
}

// ComputeCapability returns the CUDA compute capability of a device with the specified index as a string
// or an error if this cannot be determined.
func ComputeCapability(index int) (string, error) {
	lib, err := load()
	if err != nil {
		return "", err
	}
	defer lib.Close()

	if err := lib.Lookup("cuInit"); err != nil {
		return "", fmt.Errorf("failed to lookup symbol: %v", err)
	}
	if err := lib.Lookup("cuDeviceGet"); err != nil {
		return "", fmt.Errorf("failed to lookup symbol: %v", err)
	}
	if err := lib.Lookup("cuDeviceGetAttribute"); err != nil {
		return "", fmt.Errorf("failed to lookup symbol: %v", err)
	}

	if result := C.cuInit(C.uint(0)); result != C.CUDA_SUCCESS {
		return "", fmt.Errorf("failed to initialize CUDA: result=%v", result)
	}

	var device C.CUdevice
	// NOTE: We only query the first device
	if result := C.cuDeviceGet(&device, C.int(index)); result != C.CUDA_SUCCESS {
		return "", fmt.Errorf("failed to get CUDA device %v: result=%v", 0, result)
	}

	var major C.int
	if result := C.cuDeviceGetAttribute(&major, C.CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MAJOR, device); result != C.CUDA_SUCCESS {
		return "", fmt.Errorf("failed to get CUDA compute capability major for device %v : result=%v", 0, result)
	}

	var minor C.int
	if result := C.cuDeviceGetAttribute(&minor, C.CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MINOR, device); result != C.CUDA_SUCCESS {
		return "", fmt.Errorf("failed to get CUDA compute capability minor for device %v: result=%v", 0, result)
	}

	return fmt.Sprintf("%d.%d", major, minor), nil
}

func load() (*dl.DynamicLibrary, error) {
	lib := dl.New(libraryName, libraryLoadFlags)
	if lib == nil {
		return nil, fmt.Errorf("error instantiating DynamicLibrary for CUDA")
	}
	err := lib.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening DynamicLibrary for CUDA: %v", err)
	}

	return lib, nil
}
