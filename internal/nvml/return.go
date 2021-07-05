/*
 * Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nvml

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type Return interface {
	Value() nvml.Return
	String() string
	Error() string
}

type nvmlReturn nvml.Return
type MockReturn nvml.Return

var _ Return = (*nvmlReturn)(nil)
var _ Return = (*MockReturn)(nil)

func (r nvmlReturn) Value() nvml.Return {
	return nvml.Return(r)
}

func (r nvmlReturn) String() string {
	return r.Error()
}

func (r nvmlReturn) Error() string {
	return nvml.ErrorString(nvml.Return(r))
}

func (r MockReturn) Value() nvml.Return {
	return nvml.Return(r)
}

func (r MockReturn) String() string {
	return r.Error()
}

func (r MockReturn) Error() string {
	switch nvml.Return(r) {
	case SUCCESS:
		return "SUCCESS"
	case ERROR_UNINITIALIZED:
		return "ERROR_UNINITIALIZED"
	case ERROR_INVALID_ARGUMENT:
		return "ERROR_INVALID_ARGUMENT"
	case ERROR_NOT_SUPPORTED:
		return "ERROR_NOT_SUPPORTED"
	case ERROR_NO_PERMISSION:
		return "ERROR_NO_PERMISSION"
	case ERROR_ALREADY_INITIALIZED:
		return "ERROR_ALREADY_INITIALIZED"
	case ERROR_NOT_FOUND:
		return "ERROR_NOT_FOUND"
	case ERROR_INSUFFICIENT_SIZE:
		return "ERROR_INSUFFICIENT_SIZE"
	case ERROR_INSUFFICIENT_POWER:
		return "ERROR_INSUFFICIENT_POWER"
	case ERROR_DRIVER_NOT_LOADED:
		return "ERROR_DRIVER_NOT_LOADED"
	case ERROR_TIMEOUT:
		return "ERROR_TIMEOUT"
	case ERROR_IRQ_ISSUE:
		return "ERROR_IRQ_ISSUE"
	case ERROR_LIBRARY_NOT_FOUND:
		return "ERROR_LIBRARY_NOT_FOUND"
	case ERROR_FUNCTION_NOT_FOUND:
		return "ERROR_FUNCTION_NOT_FOUND"
	case ERROR_CORRUPTED_INFOROM:
		return "ERROR_CORRUPTED_INFOROM"
	case ERROR_GPU_IS_LOST:
		return "ERROR_GPU_IS_LOST"
	case ERROR_RESET_REQUIRED:
		return "ERROR_RESET_REQUIRED"
	case ERROR_OPERATING_SYSTEM:
		return "ERROR_OPERATING_SYSTEM"
	case ERROR_LIB_RM_VERSION_MISMATCH:
		return "ERROR_LIB_RM_VERSION_MISMATCH"
	case ERROR_IN_USE:
		return "ERROR_IN_USE"
	case ERROR_MEMORY:
		return "ERROR_MEMORY"
	case ERROR_NO_DATA:
		return "ERROR_NO_DATA"
	case ERROR_VGPU_ECC_NOT_SUPPORTED:
		return "ERROR_VGPU_ECC_NOT_SUPPORTED"
	case ERROR_INSUFFICIENT_RESOURCES:
		return "ERROR_INSUFFICIENT_RESOURCES"
	case ERROR_UNKNOWN:
		return "ERROR_UNKNOWN"
	default:
		return "Unknown return value"
	}
}
