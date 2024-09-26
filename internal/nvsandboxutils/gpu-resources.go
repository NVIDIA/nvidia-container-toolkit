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
	"strings"
	"unsafe"
)

import "C"

type GpuResource struct {
	Version uint32
}

type GpuFileInfo struct {
	Path    string
	Type    FileType
	SubType FileSystemSubType
	Module  FileModule
	Flags   FileFlag
}

func (l *library) GetGpuResource(uuid string) ([]GpuFileInfo, Ret) {
	deviceType := NV_GPU_INPUT_GPU_UUID
	if strings.HasPrefix(uuid, "MIG-") {
		deviceType = NV_GPU_INPUT_MIG_UUID
	}

	request := GpuRes{
		Version:   1,
		InputType: uint32(deviceType),
		Input:     convertStringToFixedArray(uuid),
	}

	ret := nvSandboxUtilsGetGpuResource(&request)
	if ret != SUCCESS {
		return nil, ret
	}

	var fileInfos []GpuFileInfo
	for fileInfo := request.Files; fileInfo != nil; fileInfo = fileInfo.Next {
		fi := GpuFileInfo{
			Path:    C.GoString((*C.char)(unsafe.Pointer(fileInfo.FilePath))),
			Type:    FileType(fileInfo.FileType),
			SubType: FileSystemSubType(fileInfo.FileSubType),
			Module:  FileModule(fileInfo.Module),
			Flags:   FileFlag(fileInfo.Flags),
		}
		fileInfos = append(fileInfos, fi)
	}
	return fileInfos, SUCCESS
}
