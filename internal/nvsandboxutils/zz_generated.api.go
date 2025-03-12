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

// Generated Code; DO NOT EDIT.

package nvsandboxutils

// The variables below represent package level methods from the library type.
var (
	ErrorString = libnvsandboxutils.ErrorString
	GetDriverVersion = libnvsandboxutils.GetDriverVersion
	GetFileContent = libnvsandboxutils.GetFileContent
	GetGpuResource = libnvsandboxutils.GetGpuResource
	Init = libnvsandboxutils.Init
	LookupSymbol = libnvsandboxutils.LookupSymbol
	Shutdown = libnvsandboxutils.Shutdown
)

// Interface represents the interface for the library type.
//
//go:generate moq -rm -fmt=goimports -out mock/interface.go -pkg mock . Interface:Interface
type Interface interface {
	ErrorString(Ret) string
	GetDriverVersion() (string, Ret)
	GetFileContent(string) (string, Ret)
	GetGpuResource(string) ([]GpuFileInfo, Ret)
	Init(string) Ret
	LookupSymbol(string) error
	Shutdown() Ret
}
