/*
 * Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package server

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/gpus"
)

// Compile-time interface checks
var _ nvml.Interface = (*Server)(nil)
var _ nvml.ExtendedInterface = (*Server)(nil)

type Option func(*options) error

func New(opts ...Option) (*Server, error) {
	o := &options{}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}
	// TODO: Check defaults and validity
	return o.build(), nil
}

// NewServerFromConfig creates a new server from the provided configuration
func (o *options) build() *Server {
	devices := make([]nvml.Device, len(o.gpus))
	for i, gpu := range o.gpus {
		devices[i] = NewDeviceFromConfig(gpu, i)
	}

	server := &Server{
		Devices:           devices,
		DriverVersion:     o.DriverVersion,
		NvmlVersion:       o.NvmlVersion,
		CudaDriverVersion: o.CudaDriverVersion,
	}
	server.SetMockFuncs()
	return server
}

// GBtoMB is a conversion constant from GB to MB (1 GB = 1024 MB)
const GBtoMB = 1024

// options contains the minimal configuration needed for a server
type options struct {
	gpus              []gpus.Config
	DriverVersion     string
	NvmlVersion       string
	CudaDriverVersion int
}

// Server provides a reusable server implementation
type Server struct {
	mock.Interface
	mock.ExtendedInterface
	Devices           []nvml.Device
	DriverVersion     string
	NvmlVersion       string
	CudaDriverVersion int
}

// Device provides a reusable device implementation
type Device struct {
	mock.Device
	sync.RWMutex
	Config             gpus.Config // Embedded configuration
	UUID               string
	PciBusID           string
	Minor              int
	Index              int
	MigMode            int
	GpuInstances       map[*GpuInstance]struct{}
	GpuInstanceCounter uint32
	MemoryInfo         nvml.Memory
}

// GpuInstance provides a reusable GPU instance implementation
type GpuInstance struct {
	mock.GpuInstance
	sync.RWMutex
	Info                   nvml.GpuInstanceInfo
	ComputeInstances       map[*ComputeInstance]struct{}
	ComputeInstanceCounter uint32
	MIGProfiles            gpus.MIGProfileConfig
}

// ComputeInstance provides a reusable compute instance implementation
type ComputeInstance struct {
	mock.ComputeInstance
	Info nvml.ComputeInstanceInfo
}

// CudaComputeCapability represents CUDA compute capability
type CudaComputeCapability struct {
	Major int
	Minor int
}

var _ nvml.Interface = (*Server)(nil)
var _ nvml.Device = (*Device)(nil)
var _ nvml.GpuInstance = (*GpuInstance)(nil)
var _ nvml.ComputeInstance = (*ComputeInstance)(nil)

// NewServerWithGPUs creates a new server with heterogeneous GPU configurations
func NewServerWithGPUs(driverVersion, nvmlVersion string, cudaDriverVersion int, gpuConfigs ...gpus.Config) *Server {
	devices := make([]nvml.Device, len(gpuConfigs))
	for i, config := range gpuConfigs {
		devices[i] = NewDeviceFromConfig(config, i)
	}

	server := &Server{
		Devices:           devices,
		DriverVersion:     driverVersion,
		NvmlVersion:       nvmlVersion,
		CudaDriverVersion: cudaDriverVersion,
	}
	server.SetMockFuncs()
	return server
}

// NewDeviceFromConfig creates a new device from the provided GPU configuration
func NewDeviceFromConfig(config gpus.Config, index int) *Device {
	device := &Device{
		Config:             config,
		UUID:               "GPU-" + uuid.New().String(),
		PciBusID:           fmt.Sprintf("0000:%02x:00.0", index),
		Minor:              index,
		Index:              index,
		GpuInstances:       make(map[*GpuInstance]struct{}),
		GpuInstanceCounter: 0,
		MemoryInfo:         nvml.Memory{Total: config.MemoryMB * 1024 * 1024, Free: 0, Used: 0},
	}
	device.SetMockFuncs()
	return device
}

// NewGpuInstanceFromInfo creates a new GPU instance
func NewGpuInstanceFromInfo(info nvml.GpuInstanceInfo, profiles gpus.MIGProfileConfig) *GpuInstance {
	gi := &GpuInstance{
		Info:                   info,
		ComputeInstances:       make(map[*ComputeInstance]struct{}),
		ComputeInstanceCounter: 0,
		MIGProfiles:            profiles,
	}
	gi.SetMockFuncs()
	return gi
}

// NewComputeInstanceFromInfo creates a new compute instance
func NewComputeInstanceFromInfo(info nvml.ComputeInstanceInfo) *ComputeInstance {
	ci := &ComputeInstance{
		Info: info,
	}
	ci.SetMockFuncs()
	return ci
}

// SetMockFuncs configures all the mock function implementations for the server
func (s *Server) SetMockFuncs() {
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

// SetMockFuncs configures all the mock function implementations for the device
func (d *Device) SetMockFuncs() {
	d.GetMinorNumberFunc = func() (int, nvml.Return) {
		return d.Minor, nvml.SUCCESS
	}

	d.GetIndexFunc = func() (int, nvml.Return) {
		return d.Index, nvml.SUCCESS
	}

	d.GetCudaComputeCapabilityFunc = func() (int, int, nvml.Return) {
		return d.Config.CudaMajor, d.Config.CudaMinor, nvml.SUCCESS
	}

	d.GetUUIDFunc = func() (string, nvml.Return) {
		return d.UUID, nvml.SUCCESS
	}

	d.GetNameFunc = func() (string, nvml.Return) {
		return d.Config.Name, nvml.SUCCESS
	}

	d.GetBrandFunc = func() (nvml.BrandType, nvml.Return) {
		return d.Config.Brand, nvml.SUCCESS
	}

	d.GetArchitectureFunc = func() (nvml.DeviceArchitecture, nvml.Return) {
		return d.Config.Architecture, nvml.SUCCESS
	}

	d.GetMemoryInfoFunc = func() (nvml.Memory, nvml.Return) {
		return d.MemoryInfo, nvml.SUCCESS
	}

	d.GetPciInfoFunc = func() (nvml.PciInfo, nvml.Return) {
		if d.Config.PciInfo != nil {
			return *d.Config.PciInfo, nvml.SUCCESS
		}
		//nolint:staticcheck
		id := d.Config.PciDeviceId
		if id == 0 {
			return nvml.PciInfo{}, nvml.ERROR_NOT_SUPPORTED
		}
		p := nvml.PciInfo{
			PciDeviceId: id,
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

		if _, exists := d.Config.MIGProfiles.GpuInstanceProfiles[giProfileId]; !exists {
			return nvml.GpuInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		return d.Config.MIGProfiles.GpuInstanceProfiles[giProfileId], nvml.SUCCESS
	}

	d.GetGpuInstancePossiblePlacementsFunc = func(info *nvml.GpuInstanceProfileInfo) ([]nvml.GpuInstancePlacement, nvml.Return) {
		return d.Config.MIGProfiles.GpuInstancePlacements[int(info.Id)], nvml.SUCCESS
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
		gi := NewGpuInstanceFromInfo(giInfo, d.Config.MIGProfiles)
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
		gi := NewGpuInstanceFromInfo(giInfo, d.Config.MIGProfiles)
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

// SetMockFuncs configures all the mock function implementations for the GPU instance
func (gi *GpuInstance) SetMockFuncs() {
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

		if _, exists := gi.MIGProfiles.ComputeInstanceProfiles[giProfileId]; !exists {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		if _, exists := gi.MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId]; !exists {
			return nvml.ComputeInstanceProfileInfo{}, nvml.ERROR_NOT_SUPPORTED
		}

		return gi.MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId], nvml.SUCCESS
	}

	gi.GetComputeInstancePossiblePlacementsFunc = func(info *nvml.ComputeInstanceProfileInfo) ([]nvml.ComputeInstancePlacement, nvml.Return) {
		return gi.MIGProfiles.ComputeInstancePlacements[int(gi.Info.Id)][int(info.Id)], nvml.SUCCESS
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
		ci := NewComputeInstanceFromInfo(ciInfo)
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

// SetMockFuncs configures all the mock function implementations for the compute instance
func (ci *ComputeInstance) SetMockFuncs() {
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
