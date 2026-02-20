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

package dl

// #cgo LDFLAGS: -ldl
// #define _GNU_SOURCE
// #include <dlfcn.h>
// #include <stdlib.h>
// #include <linux/limits.h>
import "C"
import (
	"fmt"
	"path/filepath"
	"unsafe"
)

const (
	RTLD_DEEPBIND = C.RTLD_DEEPBIND
)

// Path returns the path to the loaded library.
// See https://man7.org/linux/man-pages/man3/dlinfo.3.html
func (dl *DynamicLibrary) Path() (string, error) {
	if dl.handle == nil {
		return "", fmt.Errorf("%v not opened", dl.Name)
	}

	libParentPathBuffer := C.CBytes(make([]byte, 0, C.PATH_MAX))
	defer C.free(unsafe.Pointer(libParentPathBuffer))

	var libPath string
	if err := withOSLock(func() error {
		if dl.path != "" {
			libPath = dl.path
			return nil
		}
		// Call dlError() to clear out any previous errors.
		_ = dlError()
		ret := C.dlinfo(dl.handle, C.RTLD_DI_ORIGIN, libParentPathBuffer)
		if ret == -1 {
			return fmt.Errorf("dlinfo call failed: %w", dlError())
		}

		libPath = filepath.Join(C.GoString((*C.char)(libParentPathBuffer)), dl.Name)
		dl.path = libPath

		return nil
	}); err != nil {
		return "", err
	}
	return libPath, nil
}
