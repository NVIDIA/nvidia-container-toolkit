/*
 * Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

package dgxa100

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	"github.com/google/uuid"
)

type Server struct {
	mock.Interface
	mock.ExtendedInterface
	Devices           [8]nvml.Device
	DriverVersion     string
	NvmlVersion       string
	CudaDriverVersion int
}
type Device struct {
	mock.Device
	sync.RWMutex
	UUID                  string
	Name                  string
	Brand                 nvml.BrandType
	Architecture          nvml.DeviceArchitecture
	PciBusID              string
	Minor                 int
	Index                 int
	CudaComputeCapability CudaComputeCapability
	MigMode               int
	GpuInstances          map[*GpuInstance]struct{}
	GpuInstanceCounter    uint32
	MemoryInfo            nvml.Memory
}

type GpuInstance struct {
	mock.GpuInstance
	sync.RWMutex
	Info                   nvml.GpuInstanceInfo
	ComputeInstances       map[*ComputeInstance]struct{}
	ComputeInstanceCounter uint32
}

type ComputeInstance struct {
	mock.ComputeInstance
	Info nvml.ComputeInstanceInfo
}

type CudaComputeCapability struct {
	Major int
	Minor int
}

var _ nvml.Interface = (*Server)(nil)
var _ nvml.Device = (*Device)(nil)
var _ nvml.GpuInstance = (*GpuInstance)(nil)
var _ nvml.ComputeInstance = (*ComputeInstance)(nil)

func New() *Server {
	server := &Server{
		Devices: [8]nvml.Device{
			NewDevice(0),
			NewDevice(1),
			NewDevice(2),
			NewDevice(3),
			NewDevice(4),
			NewDevice(5),
			NewDevice(6),
			NewDevice(7),
		},
		DriverVersion:     "550.54.15",
		NvmlVersion:       "12.550.54.15",
		CudaDriverVersion: 12040,
	}
	server.setMockFuncs()
	return server
}

func NewDevice(index int) *Device {
	device := &Device{
		UUID:         "GPU-" + uuid.New().String(),
		Name:         "Mock NVIDIA A100-SXM4-40GB",
		Brand:        nvml.BRAND_NVIDIA,
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		PciBusID:     fmt.Sprintf("0000:%02x:00.0", index),
		Minor:        index,
		Index:        index,
		CudaComputeCapability: CudaComputeCapability{
			Major: 8,
			Minor: 0,
		},
		GpuInstances:       make(map[*GpuInstance]struct{}),
		GpuInstanceCounter: 0,
		MemoryInfo:         nvml.Memory{42949672960, 0, 0},
	}
	device.setMockFuncs()
	return device
}

func NewGpuInstance(info nvml.GpuInstanceInfo) *GpuInstance {
	gi := &GpuInstance{
		Info:                   info,
		ComputeInstances:       make(map[*ComputeInstance]struct{}),
		ComputeInstanceCounter: 0,
	}
	gi.setMockFuncs()
	return gi
}

func NewComputeInstance(info nvml.ComputeInstanceInfo) *ComputeInstance {
	ci := &ComputeInstance{
		Info: info,
	}
	ci.setMockFuncs()
	return ci
}

func (s *Server) setMockFuncs() {
	s.ExtensionsFunc = func() nvml.ExtendedInterface {
		return s
	}

	s.LookupSymbolFunc = func(symbol string) error {
		return nil
	}

	s.InitFunc = func() nvml.Return {
		return nvml.SUCCESS
	}

	s.ShutdownFunc = func() nvml.Return {
		return nvml.SUCCESS
	}

	s.SystemGetDriverVersionFunc = func() (string, nvml.Return) {
		return s.DriverVersion, nvml.SUCCESS
	}

	s.SystemGetNVMLVersionFunc = func() (string, nvml.Return) {
		return s.NvmlVersion, nvml.SUCCESS
	}

	s.SystemGetCudaDriverVersionFunc = func() (int, nvml.Return) {
		return s.CudaDriverVersion, nvml.SUCCESS
	}

	s.DeviceGetCountFunc = func() (int, nvml.Return) {
		return len(s.Devices), nvml.SUCCESS
	}

	s.DeviceGetHandleByIndexFunc = func(index int) (nvml.Device, nvml.Return) {
		if index < 0 || index >= len(s.Devices) {
			return nil, nvml.ERROR_INVALID_ARGUMENT
		}
		return s.Devices[index], nvml.SUCCESS
	}

	s.DeviceGetHandleByUUIDFunc = func(uuid string) (nvml.Device, nvml.Return) {
		for _, d := range s.Devices {
			if uuid == d.(*Device).UUID {
				return d, nvml.SUCCESS
			}
		}
		return nil, nvml.ERROR_INVALID_ARGUMENT
	}

	s.DeviceGetHandleByPciBusIdFunc = func(busID string) (nvml.Device, nvml.Return) {
		for _, d := range s.Devices {
			if busID == d.(*Device).PciBusID {
				return d, nvml.SUCCESS
			}
		}
		return nil, nvml.ERROR_INVALID_ARGUMENT
	}
}

func (d *Device) setMockFuncs() {
	d.GetMinorNumberFunc = func() (int, nvml.Return) {
		return d.Minor, nvml.SUCCESS
	}

	d.GetIndexFunc = func() (int, nvml.Return) {
		return d.Index, nvml.SUCCESS
	}

	d.GetCudaComputeCapabilityFunc = func() (int, int, nvml.Return) {
		return d.CudaComputeCapability.Major, d.CudaComputeCapability.Minor, nvml.SUCCESS
	}

	d.GetUUIDFunc = func() (string, nvml.Return) {
		return d.UUID, nvml.SUCCESS
	}

	d.GetNameFunc = func() (string, nvml.Return) {
		return d.Name, nvml.SUCCESS
	}

	d.GetBrandFunc = func() (nvml.BrandType, nvml.Return) {
		return d.Brand, nvml.SUCCESS
	}

	d.GetArchitectureFunc = func() (nvml.DeviceArchitecture, nvml.Return) {
		return d.Architecture, nvml.SUCCESS
	}

	d.GetMemoryInfoFunc = func() (nvml.Memory, nvml.Return) {
		return d.MemoryInfo, nvml.SUCCESS
	}

	d.GetPciInfoFunc = func() (nvml.PciInfo, nvml.Return) {
		p := nvml.PciInfo{
			PciDeviceId: 0x20B010DE,
		}
		return p, nvml.SUCCESS
	}

	d.SetMigModeFunc = func(mode int) (nvml.Return, nvml.Return) {
		d.MigMode = mode
		return nvml.SUCCESS, nvml.SUCCESS
	}

	d.GetMigModeFunc = func() (int, int, nvml.Return) {
		return d.MigMode, d.MigMode, nvml.SUCCESS
	}

	d.GetGpuInstanceProfileInfoFunc = func(giProfileId int) (nvml.GpuInstanceProfileInfo, nvml.Return) {
		if giProfileId < 0 || giProfileId >= nvml.GPU_INSTANCE_PROFILE_COUNT {
			return nvml.GpuInstanceProfileInfo{}, nvml.ERROR_INVALID_ARGUMENT
		}

		if _, exists := MIGProfiles.GpuInstanceProfiles[giProfileId]; !exists {
			return nvml.GpuInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		return MIGProfiles.GpuInstanceProfiles[giProfileId], nvml.SUCCESS
	}

	d.GetGpuInstancePossiblePlacementsFunc = func(info *nvml.GpuInstanceProfileInfo) ([]nvml.GpuInstancePlacement, nvml.Return) {
		return MIGPlacements.GpuInstancePossiblePlacements[int(info.Id)], nvml.SUCCESS
	}

	d.CreateGpuInstanceFunc = func(info *nvml.GpuInstanceProfileInfo) (nvml.GpuInstance, nvml.Return) {
		d.Lock()
		defer d.Unlock()
		giInfo := nvml.GpuInstanceInfo{
			Device:    d,
			Id:        d.GpuInstanceCounter,
			ProfileId: info.Id,
		}
		d.GpuInstanceCounter++
		gi := NewGpuInstance(giInfo)
		d.GpuInstances[gi] = struct{}{}
		return gi, nvml.SUCCESS
	}

	d.CreateGpuInstanceWithPlacementFunc = func(info *nvml.GpuInstanceProfileInfo, placement *nvml.GpuInstancePlacement) (nvml.GpuInstance, nvml.Return) {
		d.Lock()
		defer d.Unlock()
		giInfo := nvml.GpuInstanceInfo{
			Device:    d,
			Id:        d.GpuInstanceCounter,
			ProfileId: info.Id,
			Placement: *placement,
		}
		d.GpuInstanceCounter++
		gi := NewGpuInstance(giInfo)
		d.GpuInstances[gi] = struct{}{}
		return gi, nvml.SUCCESS
	}

	d.GetGpuInstancesFunc = func(info *nvml.GpuInstanceProfileInfo) ([]nvml.GpuInstance, nvml.Return) {
		d.RLock()
		defer d.RUnlock()
		var gis []nvml.GpuInstance
		for gi := range d.GpuInstances {
			if gi.Info.ProfileId == info.Id {
				gis = append(gis, gi)
			}
		}
		return gis, nvml.SUCCESS
	}
}

func (gi *GpuInstance) setMockFuncs() {
	gi.GetInfoFunc = func() (nvml.GpuInstanceInfo, nvml.Return) {
		return gi.Info, nvml.SUCCESS
	}

	gi.GetComputeInstanceProfileInfoFunc = func(ciProfileId int, ciEngProfileId int) (nvml.ComputeInstanceProfileInfo, nvml.Return) {
		if ciProfileId < 0 || ciProfileId >= nvml.COMPUTE_INSTANCE_PROFILE_COUNT {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_INVALID_ARGUMENT
		}

		if ciEngProfileId != nvml.COMPUTE_INSTANCE_ENGINE_PROFILE_SHARED {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		giProfileId := int(gi.Info.ProfileId)

		if _, exists := MIGProfiles.ComputeInstanceProfiles[giProfileId]; !exists {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		if _, exists := MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId]; !exists {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		return MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId], nvml.SUCCESS
	}

	gi.GetComputeInstancePossiblePlacementsFunc = func(info *nvml.ComputeInstanceProfileInfo) ([]nvml.ComputeInstancePlacement, nvml.Return) {
		return MIGPlacements.ComputeInstancePossiblePlacements[int(gi.Info.Id)][int(info.Id)], nvml.SUCCESS
	}

	gi.CreateComputeInstanceFunc = func(info *nvml.ComputeInstanceProfileInfo) (nvml.ComputeInstance, nvml.Return) {
		gi.Lock()
		defer gi.Unlock()
		ciInfo := nvml.ComputeInstanceInfo{
			Device:      gi.Info.Device,
			GpuInstance: gi,
			Id:          gi.ComputeInstanceCounter,
			ProfileId:   info.Id,
		}
		gi.ComputeInstanceCounter++
		ci := NewComputeInstance(ciInfo)
		gi.ComputeInstances[ci] = struct{}{}
		return ci, nvml.SUCCESS
	}

	gi.GetComputeInstancesFunc = func(info *nvml.ComputeInstanceProfileInfo) ([]nvml.ComputeInstance, nvml.Return) {
		gi.RLock()
		defer gi.RUnlock()
		var cis []nvml.ComputeInstance
		for ci := range gi.ComputeInstances {
			if ci.Info.ProfileId == info.Id {
				cis = append(cis, ci)
			}
		}
		return cis, nvml.SUCCESS
	}

	gi.DestroyFunc = func() nvml.Return {
		d := gi.Info.Device.(*Device)
		d.Lock()
		defer d.Unlock()
		delete(d.GpuInstances, gi)
		return nvml.SUCCESS
	}
}

func (ci *ComputeInstance) setMockFuncs() {
	ci.GetInfoFunc = func() (nvml.ComputeInstanceInfo, nvml.Return) {
		return ci.Info, nvml.SUCCESS
	}

	ci.DestroyFunc = func() nvml.Return {
		gi := ci.Info.GpuInstance.(*GpuInstance)
		gi.Lock()
		defer gi.Unlock()
		delete(gi.ComputeInstances, ci)
		return nvml.SUCCESS
	}
}
