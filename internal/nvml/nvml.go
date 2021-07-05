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

type nvmlLib struct{}
type nvmlDevice nvml.Device
type nvmlGpuInstance nvml.GpuInstance
type nvmlComputeInstance nvml.ComputeInstance

var _ Interface = (*nvmlLib)(nil)
var _ Device = (*nvmlDevice)(nil)
var _ GpuInstance = (*nvmlGpuInstance)(nil)
var _ ComputeInstance = (*nvmlComputeInstance)(nil)

func New() Interface {
	return &nvmlLib{}
}

func (n *nvmlLib) Init() Return {
	return nvmlReturn(nvml.Init())
}

func (n *nvmlLib) Shutdown() Return {
	return nvmlReturn(nvml.Shutdown())
}

func (n *nvmlLib) DeviceGetCount() (int, Return) {
	c, r := nvml.DeviceGetCount()
	return c, nvmlReturn(r)
}

func (n *nvmlLib) DeviceGetHandleByIndex(index int) (Device, Return) {
	d, r := nvml.DeviceGetHandleByIndex(index)
	return nvmlDevice(d), nvmlReturn(r)
}

func (n *nvmlLib) SystemGetDriverVersion() (string, Return) {
	v, r := nvml.SystemGetDriverVersion()
	return v, nvmlReturn(r)
}

func (d nvmlDevice) GetIndex() (int, Return) {
	i, r := nvml.Device(d).GetIndex()
	return i, nvmlReturn(r)
}

func (d nvmlDevice) GetPciInfo() (PciInfo, Return) {
	p, r := nvml.Device(d).GetPciInfo()
	return PciInfo(p), nvmlReturn(r)
}

func (d nvmlDevice) GetUUID() (string, Return) {
	u, r := nvml.Device(d).GetUUID()
	return u, nvmlReturn(r)
}

func (d nvmlDevice) GetMinorNumber() (int, Return) {
	m, r := nvml.Device(d).GetMinorNumber()
	return m, nvmlReturn(r)
}

func (d nvmlDevice) IsMigDeviceHandle() (bool, Return) {
	b, r := nvml.Device(d).IsMigDeviceHandle()
	return b, nvmlReturn(r)
}

func (d nvmlDevice) GetDeviceHandleFromMigDeviceHandle() (Device, Return) {
	p, r := nvml.Device(d).GetDeviceHandleFromMigDeviceHandle()
	return nvmlDevice(p), nvmlReturn(r)
}

func (d nvmlDevice) GetGPUInstanceId() (int, Return) {
	gi, r := nvml.Device(d).GetGpuInstanceId()
	return gi, nvmlReturn(r)
}

func (d nvmlDevice) GetComputeInstanceId() (int, Return) {
	ci, r := nvml.Device(d).GetComputeInstanceId()
	return ci, nvmlReturn(r)
}

func (d nvmlDevice) SetMigMode(mode int) (Return, Return) {
	r1, r2 := nvml.Device(d).SetMigMode(mode)
	return nvmlReturn(r1), nvmlReturn(r2)
}

func (d nvmlDevice) GetMigMode() (int, int, Return) {
	s1, s2, r := nvml.Device(d).GetMigMode()
	return s1, s2, nvmlReturn(r)
}

func (d nvmlDevice) GetGpuInstanceProfileInfo(profile int) (GpuInstanceProfileInfo, Return) {
	p, r := nvml.Device(d).GetGpuInstanceProfileInfo(profile)
	return GpuInstanceProfileInfo(p), nvmlReturn(r)
}

func (d nvmlDevice) CreateGpuInstance(info *GpuInstanceProfileInfo) (GpuInstance, Return) {
	gi, r := nvml.Device(d).CreateGpuInstance((*nvml.GpuInstanceProfileInfo)(info))
	return nvmlGpuInstance(gi), nvmlReturn(r)
}

func (d nvmlDevice) GetGpuInstances(info *GpuInstanceProfileInfo) ([]GpuInstance, Return) {
	nvmlGis, r := nvml.Device(d).GetGpuInstances((*nvml.GpuInstanceProfileInfo)(info))
	var gis []GpuInstance
	for _, gi := range nvmlGis {
		gis = append(gis, nvmlGpuInstance(gi))
	}
	return gis, nvmlReturn(r)
}

func (d nvmlDevice) GetMaxMigDeviceCount() (int, Return) {
	m, r := nvml.Device(d).GetMaxMigDeviceCount()
	return m, nvmlReturn(r)
}

func (d nvmlDevice) GetMigDeviceHandleByIndex(Index int) (Device, Return) {
	h, r := nvml.Device(d).GetMigDeviceHandleByIndex(Index)
	return nvmlDevice(h), nvmlReturn(r)
}

func (gi nvmlGpuInstance) GetInfo() (GpuInstanceInfo, Return) {
	i, r := nvml.GpuInstance(gi).GetInfo()
	info := GpuInstanceInfo{
		Device:    nvmlDevice(i.Device),
		Id:        i.Id,
		ProfileId: i.ProfileId,
		Placement: i.Placement,
	}
	return info, nvmlReturn(r)
}

func (gi nvmlGpuInstance) GetComputeInstanceProfileInfo(profile int, engProfile int) (ComputeInstanceProfileInfo, Return) {
	p, r := nvml.GpuInstance(gi).GetComputeInstanceProfileInfo(profile, engProfile)
	return ComputeInstanceProfileInfo(p), nvmlReturn(r)
}

func (gi nvmlGpuInstance) CreateComputeInstance(info *ComputeInstanceProfileInfo) (ComputeInstance, Return) {
	ci, r := nvml.GpuInstance(gi).CreateComputeInstance((*nvml.ComputeInstanceProfileInfo)(info))
	return nvmlComputeInstance(ci), nvmlReturn(r)
}

func (gi nvmlGpuInstance) GetComputeInstances(info *ComputeInstanceProfileInfo) ([]ComputeInstance, Return) {
	nvmlCis, r := nvml.GpuInstance(gi).GetComputeInstances((*nvml.ComputeInstanceProfileInfo)(info))
	var cis []ComputeInstance
	for _, ci := range nvmlCis {
		cis = append(cis, nvmlComputeInstance(ci))
	}
	return cis, nvmlReturn(r)
}

func (gi nvmlGpuInstance) Destroy() Return {
	r := nvml.GpuInstance(gi).Destroy()
	return nvmlReturn(r)
}

func (ci nvmlComputeInstance) GetInfo() (ComputeInstanceInfo, Return) {
	i, r := nvml.ComputeInstance(ci).GetInfo()
	info := ComputeInstanceInfo{
		Device:      nvmlDevice(i.Device),
		GpuInstance: nvmlGpuInstance(i.GpuInstance),
		Id:          i.Id,
		ProfileId:   i.ProfileId,
	}
	return info, nvmlReturn(r)
}

func (ci nvmlComputeInstance) Destroy() Return {
	r := nvml.ComputeInstance(ci).Destroy()
	return nvmlReturn(r)
}
