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
	"unsafe"
)

/*
#include <stdlib.h>
*/
import "C"

var cgoAllocsUnknown = new(struct{})

type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

func uint32SliceToIntSlice(s []uint32) []int {
	ret := make([]int, len(s))
	for i := range s {
		ret[i] = int(s[i])
	}
	return ret
}

func convertSlice[T any, I any](input []T) []I {
	output := make([]I, len(input))
	for i, obj := range input {
		switch v := any(obj).(type) {
		case I:
			output[i] = v
		}
	}
	return output
}

func int32SliceToMask255(s []int32) Mask255 {
	var m Mask255
	for _, p := range s {
		if p < 0 || p >= 255 {
			continue
		}
		m.Mask[p/32] |= 1 << (uint32(p) % 32)
	}
	return m
}

// packPCharString creates a Go string backed by *C.char and avoids copying.
func packPCharString(p *C.char) (raw string) {
	if p != nil && *p != 0 {
		h := (*stringHeader)(unsafe.Pointer(&raw))
		h.Data = unsafe.Pointer(p)
		for *p != 0 {
			p = (*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 1)) // p++
		}
		h.Len = int(uintptr(unsafe.Pointer(p)) - uintptr(h.Data))
	}
	return
}

// unpackPCharString represents the data from Go string as *C.char and avoids copying.
func unpackPCharString(str string) (*C.char, *struct{}) {
	h := (*stringHeader)(unsafe.Pointer(&str))
	return (*C.char)(h.Data), cgoAllocsUnknown
}

func malloc(size uintptr) unsafe.Pointer {
	return C.malloc(C.size_t(size))
}

func free(ptr unsafe.Pointer) {
	C.free(ptr)
}

// int8SliceToString converts a NUL-terminated C char array (typed as []int8)
// into a Go string, stopping at the first NUL.
func int8SliceToString(s []int8) string {
	buf := make([]byte, len(s))
	for i, c := range s {
		buf[i] = byte(c)
	}
	return string(buf[:clen(buf)])
}

// stringToInt8Slice copies s into out as a NUL-terminated C string. At most
// len(out)-1 bytes are written so the final byte is always a NUL terminator;
// remaining bytes in out are zeroed.
func stringToInt8Slice(s string, out []int8) {
	n := len(s)
	if n > len(out)-1 {
		n = len(out) - 1
	}
	for i := 0; i < n; i++ {
		out[i] = int8(s[i])
	}
	for i := n; i < len(out); i++ {
		out[i] = 0
	}
}
