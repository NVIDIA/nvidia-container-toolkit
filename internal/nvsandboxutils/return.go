/**
# Copyright 2024 NVIDIA CORPORATION
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

package nvsandboxutils

import (
	"fmt"
)

// nvsandboxutils.ErrorString()
func (l *library) ErrorString(r Ret) string {
	return r.Error()
}

// String returns the string representation of a Ret.
func (r Ret) String() string {
	return r.Error()
}

// Error returns the string representation of a Ret.
func (r Ret) Error() string {
	return errorStringFunc(r)
}

// Assigned to nvsandboxutils.ErrorString if the system nvsandboxutils library is in use.
var errorStringFunc = defaultErrorStringFunc

// nvsanboxutilsErrorString is an alias for the default error string function.
var nvsanboxutilsErrorString = defaultErrorStringFunc

// defaultErrorStringFunc provides a basic nvsandboxutils.ErrorString implementation.
// This allows the nvsandboxutils.ErrorString function to be used even if the nvsandboxutils library
// is not loaded.
var defaultErrorStringFunc = func(r Ret) string {
	switch r {
	case SUCCESS:
		return "SUCCESS"
	case ERROR_UNINITIALIZED:
		return "ERROR_UNINITIALIZED"
	case ERROR_NOT_SUPPORTED:
		return "ERROR_NOT_SUPPORTED"
	case ERROR_INVALID_ARG:
		return "ERROR_INVALID_ARG"
	case ERROR_INSUFFICIENT_SIZE:
		return "ERROR_INSUFFICIENT_SIZE"
	case ERROR_VERSION_NOT_SUPPORTED:
		return "ERROR_VERSION_NOT_SUPPORTED"
	case ERROR_LIBRARY_LOAD:
		return "ERROR_LIBRARY_LOAD"
	case ERROR_FUNCTION_NOT_FOUND:
		return "ERROR_FUNCTION_NOT_FOUND"
	case ERROR_DEVICE_NOT_FOUND:
		return "ERROR_DEVICE_NOT_FOUND"
	case ERROR_NVML_LIB_CALL:
		return "ERROR_NVML_LIB_CALL"
	case ERROR_UNKNOWN:
		return "ERROR_UNKNOWN"
	default:
		return fmt.Sprintf("unknown return value: %d", r)
	}
}
