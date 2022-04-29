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

typedef enum cudaError_enum {
	CUDA_SUCCESS = 0
} CUresult;

CUresult CUDAAPI cuDriverGetVersion(int *driverVersion);
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
