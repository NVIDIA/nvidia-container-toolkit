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

type Interface interface {
	Init() Return
	Shutdown() Return
	DeviceGetCount() (int, Return)
	DeviceGetHandleByIndex(Index int) (Device, Return)
	SystemGetDriverVersion() (string, Return)
}

type Device interface {
	GetIndex() (int, Return)
	GetPciInfo() (PciInfo, Return)
	GetUUID() (string, Return)
	GetMinorNumber() (int, Return)
	IsMigDeviceHandle() (bool, Return)
	GetDeviceHandleFromMigDeviceHandle() (Device, Return)
	SetMigMode(Mode int) (Return, Return)
	GetMigMode() (int, int, Return)
	GetGpuInstanceProfileInfo(Profile int) (GpuInstanceProfileInfo, Return)
	CreateGpuInstance(Info *GpuInstanceProfileInfo) (GpuInstance, Return)
	GetGpuInstances(Info *GpuInstanceProfileInfo) ([]GpuInstance, Return)
	GetMaxMigDeviceCount() (int, Return)
	GetMigDeviceHandleByIndex(Index int) (Device, Return)
	GetGPUInstanceId() (int, Return)
	GetComputeInstanceId() (int, Return)
}

type GpuInstance interface {
	GetInfo() (GpuInstanceInfo, Return)
	GetComputeInstanceProfileInfo(Profile int, EngProfile int) (ComputeInstanceProfileInfo, Return)
	CreateComputeInstance(Info *ComputeInstanceProfileInfo) (ComputeInstance, Return)
	GetComputeInstances(Info *ComputeInstanceProfileInfo) ([]ComputeInstance, Return)
	Destroy() Return
}

type ComputeInstance interface {
	GetInfo() (ComputeInstanceInfo, Return)
	Destroy() Return
}

type GpuInstanceInfo struct {
	Device    Device
	Id        uint32
	ProfileId uint32
	Placement nvml.GpuInstancePlacement
}

type ComputeInstanceInfo struct {
	Device      Device
	GpuInstance GpuInstance
	Id          uint32
	ProfileId   uint32
}

type PciInfo nvml.PciInfo
type GpuInstanceProfileInfo nvml.GpuInstanceProfileInfo
type ComputeInstanceProfileInfo nvml.ComputeInstanceProfileInfo
