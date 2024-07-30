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

import "C"

func (l *library) Init(path string) Ret {
	if err := l.load(); err != nil {
		return ERROR_LIBRARY_LOAD
	}

	input := InitInput{
		Version: 1,
		Type:    uint32(NV_ROOTFS_PATH),
		Value:   convertStringToFixedArray(path),
	}

	return nvSandboxUtilsInit(&input)
}

func (l *library) Shutdown() Ret {
	ret := nvSandboxUtilsShutdown()
	if ret != SUCCESS {
		return ret
	}

	err := l.close()
	if err != nil {
		return ERROR_UNKNOWN
	}

	return ret
}

// TODO: Is this length specified in the header file?
const VERSION_LENGTH = 100

func (l *library) GetDriverVersion() (string, Ret) {
	Version := make([]byte, VERSION_LENGTH)
	ret := nvSandboxUtilsGetDriverVersion(&Version[0], VERSION_LENGTH)
	return string(Version[:clen(Version)]), ret
}

func (l *library) GetFileContent(path string) (string, Ret) {
	Content := make([]byte, MAX_FILE_PATH)
	FilePath := []byte(path + string(byte(0)))
	Size := uint32(MAX_FILE_PATH)
	ret := nvSandboxUtilsGetFileContent(&FilePath[0], &Content[0], &Size)
	return string(Content[:clen(Content)]), ret
}
