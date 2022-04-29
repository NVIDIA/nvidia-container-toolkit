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

package dl

import (
	"fmt"
	"unsafe"
)

// #cgo LDFLAGS: -ldl
// #include <dlfcn.h>
// #include <stdlib.h>
import "C"

const (
	RTLD_LAZY = C.RTLD_LAZY
	RTLD_NOW = C.RTLD_NOW
	RTLD_GLOBAL = C.RTLD_GLOBAL
	RTLD_LOCAL = C.RTLD_LOCAL
	RTLD_NODELETE = C.RTLD_NODELETE
	RTLD_NOLOAD = C.RTLD_NOLOAD
	RTLD_DEEPBIND = C.RTLD_DEEPBIND
)

type DynamicLibrary struct{
	Name string
	Flags int
	handle unsafe.Pointer
}

func New(name string, flags int) *DynamicLibrary {
	return &DynamicLibrary{
		Name: name,
		Flags: flags,
		handle: nil,
    }
}

func (dl *DynamicLibrary) Open() error {
	name := C.CString(dl.Name)
	defer C.free(unsafe.Pointer(name))

	handle := C.dlopen(name, C.int(dl.Flags))
	if handle == C.NULL {
		return fmt.Errorf("%s", C.GoString(C.dlerror()))
	}
	dl.handle = handle
	return nil
}

func (dl *DynamicLibrary) Close() error {
	err := C.dlclose(dl.handle)
	if err != 0 {
		return fmt.Errorf("%s", C.GoString(C.dlerror()))
	}
	return nil
}

func (dl *DynamicLibrary) Lookup(symbol string) error {
	sym := C.CString(symbol)
	defer C.free(unsafe.Pointer(sym))

	C.dlerror() // Clear out any previous errors
	C.dlsym(dl.handle, sym)
	err := C.dlerror()
	if unsafe.Pointer(err) == C.NULL {
		return nil
	}
	return fmt.Errorf("%s", C.GoString(err))
}
