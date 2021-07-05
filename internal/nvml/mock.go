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

import "fmt"

type MockServer struct {
	Devices []Device
}
type MockLunaServer struct {
	MockServer
}
type MockA100Device struct {
	Index              int
	MinorNumber        int
	MigMode            int
	GpuInstances       map[*MockA100GpuInstance]struct{}
	GpuInstanceCounter uint32
}

type MockA100GpuInstance struct {
	Info                   GpuInstanceInfo
	ComputeInstances       map[*MockA100ComputeInstance]struct{}
	ComputeInstanceCounter uint32
}
type MockA100ComputeInstance struct {
	Info ComputeInstanceInfo
}

var _ Interface = (*MockLunaServer)(nil)
var _ Device = (*MockA100Device)(nil)
var _ GpuInstance = (*MockA100GpuInstance)(nil)
var _ ComputeInstance = (*MockA100ComputeInstance)(nil)

var MockA100MIGProfiles = struct {
	GpuInstanceProfiles     map[int]GpuInstanceProfileInfo
	ComputeInstanceProfiles map[int]map[int]ComputeInstanceProfileInfo
}{
	GpuInstanceProfiles: map[int]GpuInstanceProfileInfo{
		GPU_INSTANCE_PROFILE_1_SLICE: {
			Id:                  GPU_INSTANCE_PROFILE_1_SLICE,
			IsP2pSupported:      0,
			SliceCount:          1,
			InstanceCount:       7,
			MultiprocessorCount: 1,
			CopyEngineCount:     1,
			DecoderCount:        0,
			EncoderCount:        0,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        5120,
		},
		GPU_INSTANCE_PROFILE_2_SLICE: {
			Id:                  GPU_INSTANCE_PROFILE_2_SLICE,
			IsP2pSupported:      0,
			SliceCount:          2,
			InstanceCount:       3,
			MultiprocessorCount: 2,
			CopyEngineCount:     2,
			DecoderCount:        1,
			EncoderCount:        1,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        10240,
		},
		GPU_INSTANCE_PROFILE_3_SLICE: {
			Id:                  GPU_INSTANCE_PROFILE_3_SLICE,
			IsP2pSupported:      0,
			SliceCount:          3,
			InstanceCount:       2,
			MultiprocessorCount: 3,
			CopyEngineCount:     4,
			DecoderCount:        2,
			EncoderCount:        2,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        20480,
		},
		GPU_INSTANCE_PROFILE_4_SLICE: {
			Id:                  GPU_INSTANCE_PROFILE_4_SLICE,
			IsP2pSupported:      0,
			SliceCount:          4,
			InstanceCount:       1,
			MultiprocessorCount: 4,
			CopyEngineCount:     4,
			DecoderCount:        2,
			EncoderCount:        2,
			JpegCount:           0,
			OfaCount:            0,
			MemorySizeMB:        20480,
		},
		GPU_INSTANCE_PROFILE_7_SLICE: {
			Id:                  GPU_INSTANCE_PROFILE_7_SLICE,
			IsP2pSupported:      0,
			SliceCount:          7,
			InstanceCount:       1,
			MultiprocessorCount: 7,
			CopyEngineCount:     8,
			DecoderCount:        5,
			EncoderCount:        5,
			JpegCount:           1,
			OfaCount:            1,
			MemorySizeMB:        40960,
		},
	},
	ComputeInstanceProfiles: map[int]map[int]ComputeInstanceProfileInfo{
		GPU_INSTANCE_PROFILE_1_SLICE: {
			COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         1,
				MultiprocessorCount:   1,
				SharedCopyEngineCount: 1,
				SharedDecoderCount:    0,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		GPU_INSTANCE_PROFILE_2_SLICE: {
			COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         2,
				MultiprocessorCount:   1,
				SharedCopyEngineCount: 2,
				SharedDecoderCount:    1,
				SharedEncoderCount:    1,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         1,
				MultiprocessorCount:   2,
				SharedCopyEngineCount: 2,
				SharedDecoderCount:    1,
				SharedEncoderCount:    1,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		GPU_INSTANCE_PROFILE_3_SLICE: {
			COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         3,
				MultiprocessorCount:   1,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    1,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         1,
				MultiprocessorCount:   2,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    2,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			COMPUTE_INSTANCE_PROFILE_3_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_3_SLICE,
				SliceCount:            3,
				InstanceCount:         1,
				MultiprocessorCount:   3,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    0,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		GPU_INSTANCE_PROFILE_4_SLICE: {
			COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         4,
				MultiprocessorCount:   1,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    2,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         2,
				MultiprocessorCount:   2,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    2,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
			COMPUTE_INSTANCE_PROFILE_4_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_4_SLICE,
				SliceCount:            4,
				InstanceCount:         1,
				MultiprocessorCount:   4,
				SharedCopyEngineCount: 4,
				SharedDecoderCount:    2,
				SharedEncoderCount:    2,
				SharedJpegCount:       0,
				SharedOfaCount:        0,
			},
		},
		GPU_INSTANCE_PROFILE_7_SLICE: {
			COMPUTE_INSTANCE_PROFILE_1_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_1_SLICE,
				SliceCount:            1,
				InstanceCount:         7,
				MultiprocessorCount:   1,
				SharedCopyEngineCount: 8,
				SharedDecoderCount:    5,
				SharedEncoderCount:    5,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			COMPUTE_INSTANCE_PROFILE_2_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_2_SLICE,
				SliceCount:            2,
				InstanceCount:         3,
				MultiprocessorCount:   2,
				SharedCopyEngineCount: 8,
				SharedDecoderCount:    5,
				SharedEncoderCount:    5,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			COMPUTE_INSTANCE_PROFILE_3_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_3_SLICE,
				SliceCount:            3,
				InstanceCount:         2,
				MultiprocessorCount:   3,
				SharedCopyEngineCount: 8,
				SharedDecoderCount:    5,
				SharedEncoderCount:    5,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			COMPUTE_INSTANCE_PROFILE_4_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_4_SLICE,
				SliceCount:            4,
				InstanceCount:         1,
				MultiprocessorCount:   4,
				SharedCopyEngineCount: 8,
				SharedDecoderCount:    5,
				SharedEncoderCount:    5,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
			COMPUTE_INSTANCE_PROFILE_7_SLICE: {
				Id:                    COMPUTE_INSTANCE_PROFILE_7_SLICE,
				SliceCount:            7,
				InstanceCount:         1,
				MultiprocessorCount:   7,
				SharedCopyEngineCount: 8,
				SharedDecoderCount:    5,
				SharedEncoderCount:    5,
				SharedJpegCount:       1,
				SharedOfaCount:        1,
			},
		},
	},
}

func NewMockNVMLServer(devices ...Device) Interface {
	return &MockServer{
		Devices: devices,
	}
}

func NewMockNVMLOnLunaServer() Interface {
	devices := []Device{
		NewMockA100Device(0),
		NewMockA100Device(1),
		NewMockA100Device(2),
		NewMockA100Device(3),
		NewMockA100Device(4),
		NewMockA100Device(5),
		NewMockA100Device(6),
		NewMockA100Device(7),
	}
	return NewMockNVMLServer(devices...)
}

func NewMockA100Device(index int) Device {
	return &MockA100Device{
		Index:              index,
		GpuInstances:       make(map[*MockA100GpuInstance]struct{}),
		GpuInstanceCounter: 0,
	}
}

func NewMockA100GpuInstance(info GpuInstanceInfo) GpuInstance {
	return &MockA100GpuInstance{
		Info:                   info,
		ComputeInstances:       make(map[*MockA100ComputeInstance]struct{}),
		ComputeInstanceCounter: 0,
	}
}

func NewMockA100ComputeInstance(info ComputeInstanceInfo) ComputeInstance {
	return &MockA100ComputeInstance{
		Info: info,
	}
}

func (n *MockServer) Init() Return {
	return MockReturn(SUCCESS)
}

func (n *MockServer) Shutdown() Return {
	return MockReturn(SUCCESS)
}

func (n *MockServer) DeviceGetCount() (int, Return) {
	return len(n.Devices), MockReturn(SUCCESS)
}

func (n *MockServer) DeviceGetHandleByIndex(index int) (Device, Return) {
	if index < 0 || index >= len(n.Devices) {
		return nil, MockReturn(ERROR_INVALID_ARGUMENT)
	}
	return n.Devices[index], MockReturn(SUCCESS)
}

func (n *MockServer) SystemGetDriverVersion() (string, Return) {
	return "999.99", MockReturn(SUCCESS)
}

func (d *MockA100Device) GetIndex() (int, Return) {
	return d.Index, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetPciInfo() (PciInfo, Return) {
	var busID [32]int8
	for i, b := range []byte("0000FFFF:FF:FF.F") {
		busID[i] = int8(b)
	}
	p := PciInfo{
		BusId:       busID,
		PciDeviceId: 0x20B010DE,
	}
	return p, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetUUID() (string, Return) {
	return fmt.Sprintf("GPU-%d", d.Index), MockReturn(SUCCESS)
}

func (d *MockA100Device) GetMinorNumber() (int, Return) {
	return d.MinorNumber, MockReturn(SUCCESS)
}

func (d *MockA100Device) SetMigMode(mode int) (Return, Return) {
	d.MigMode = mode
	return MockReturn(SUCCESS), MockReturn(SUCCESS)
}

func (d *MockA100Device) GetMigMode() (int, int, Return) {
	return d.MigMode, d.MigMode, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetGpuInstanceProfileInfo(giProfileId int) (GpuInstanceProfileInfo, Return) {
	if giProfileId < 0 || giProfileId >= GPU_INSTANCE_PROFILE_COUNT {
		return GpuInstanceProfileInfo{}, MockReturn(ERROR_INVALID_ARGUMENT)
	}

	if _, exists := MockA100MIGProfiles.GpuInstanceProfiles[giProfileId]; !exists {
		return GpuInstanceProfileInfo{}, MockReturn(ERROR_NOT_SUPPORTED)
	}

	return MockA100MIGProfiles.GpuInstanceProfiles[giProfileId], MockReturn(SUCCESS)
}

func (d *MockA100Device) CreateGpuInstance(info *GpuInstanceProfileInfo) (GpuInstance, Return) {
	giInfo := GpuInstanceInfo{
		Device:    d,
		Id:        d.GpuInstanceCounter,
		ProfileId: info.Id,
	}
	d.GpuInstanceCounter++
	gi := NewMockA100GpuInstance(giInfo)
	d.GpuInstances[gi.(*MockA100GpuInstance)] = struct{}{}
	return gi, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetGpuInstances(info *GpuInstanceProfileInfo) ([]GpuInstance, Return) {
	var gis []GpuInstance
	for gi := range d.GpuInstances {
		if gi.Info.ProfileId == info.Id {
			gis = append(gis, gi)
		}
	}
	return gis, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetMaxMigDeviceCount() (int, Return) {
	var count int
	for gi := range d.GpuInstances {
		count = count + int(gi.ComputeInstanceCounter)
	}
	return count, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetMigDeviceHandleByIndex(Index int) (Device, Return) {
	var count int
	for gi := range d.GpuInstances {
		if count+int(gi.ComputeInstanceCounter) < Index {
			count = count + int(gi.ComputeInstanceCounter)
			continue
		}
		for ci := range gi.ComputeInstances {
			if count < Index {
				count++
				continue
			}

			return ci, MockReturn(SUCCESS)
		}
	}
	return nil, MockReturn(ERROR_NOT_FOUND)
}

func (d *MockA100Device) GetDeviceHandleFromMigDeviceHandle() (Device, Return) {
	return nil, MockReturn(ERROR_NOT_SUPPORTED)
}

func (d *MockA100Device) IsMigDeviceHandle() (bool, Return) {
	return false, MockReturn(SUCCESS)
}

func (d *MockA100Device) GetComputeInstanceId() (int, Return) {
	panic("Not implemented: GetComputeInstanceId")
}

func (d *MockA100Device) GetGPUInstanceId() (int, Return) {
	panic("Not implemented: GetGPUInstanceId")
}

func (gi *MockA100GpuInstance) GetInfo() (GpuInstanceInfo, Return) {
	return gi.Info, MockReturn(SUCCESS)
}

func (gi *MockA100GpuInstance) GetComputeInstanceProfileInfo(ciProfileId int, ciEngProfileId int) (ComputeInstanceProfileInfo, Return) {
	if ciProfileId < 0 || ciProfileId >= COMPUTE_INSTANCE_PROFILE_COUNT {
		return ComputeInstanceProfileInfo{}, MockReturn(ERROR_INVALID_ARGUMENT)
	}

	if ciEngProfileId != COMPUTE_INSTANCE_ENGINE_PROFILE_SHARED {
		return ComputeInstanceProfileInfo{}, MockReturn(ERROR_NOT_SUPPORTED)
	}

	giProfileId := int(gi.Info.ProfileId)

	if _, exists := MockA100MIGProfiles.ComputeInstanceProfiles[giProfileId]; !exists {
		return ComputeInstanceProfileInfo{}, MockReturn(ERROR_NOT_SUPPORTED)
	}

	if _, exists := MockA100MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId]; !exists {
		return ComputeInstanceProfileInfo{}, MockReturn(ERROR_NOT_SUPPORTED)
	}

	return MockA100MIGProfiles.ComputeInstanceProfiles[giProfileId][ciProfileId], MockReturn(SUCCESS)
}

func (gi *MockA100GpuInstance) CreateComputeInstance(info *ComputeInstanceProfileInfo) (ComputeInstance, Return) {
	ciInfo := ComputeInstanceInfo{
		Device:      gi.Info.Device,
		GpuInstance: gi,
		Id:          gi.ComputeInstanceCounter,
		ProfileId:   info.Id,
	}
	gi.ComputeInstanceCounter++
	ci := NewMockA100ComputeInstance(ciInfo)
	gi.ComputeInstances[ci.(*MockA100ComputeInstance)] = struct{}{}
	return ci, MockReturn(SUCCESS)
}

func (gi *MockA100GpuInstance) GetComputeInstances(info *ComputeInstanceProfileInfo) ([]ComputeInstance, Return) {
	var cis []ComputeInstance
	for ci := range gi.ComputeInstances {
		if ci.Info.ProfileId == info.Id {
			cis = append(cis, ci)
		}
	}
	return cis, MockReturn(SUCCESS)
}

func (gi *MockA100GpuInstance) Destroy() Return {
	delete(gi.Info.Device.(*MockA100Device).GpuInstances, gi)
	return MockReturn(SUCCESS)
}

func (ci *MockA100ComputeInstance) GetInfo() (ComputeInstanceInfo, Return) {
	return ci.Info, MockReturn(SUCCESS)
}

func (ci *MockA100ComputeInstance) Destroy() Return {
	delete(ci.Info.GpuInstance.(*MockA100GpuInstance).ComputeInstances, ci)
	return MockReturn(SUCCESS)
}

// Since a compute instance can be used as a MIG device handle, it must also
// implement the Device interface
var _ Device = (*MockA100ComputeInstance)(nil)

func (c *MockA100ComputeInstance) GetIndex() (int, Return) {
	return int(c.Info.Id), MockReturn(SUCCESS)
}

func (c *MockA100ComputeInstance) GetPciInfo() (PciInfo, Return) {
	// TODO: How does this behave on an actual MIG system?
	panic("Not implemented: GetPciInfo")
}

func (c *MockA100ComputeInstance) GetUUID() (string, Return) {
	return fmt.Sprintf("MIG-%d", c.Info.Id), MockReturn(SUCCESS)
}

func (c *MockA100ComputeInstance) GetMinorNumber() (int, Return) {
	// TODO: This depends on the content of the mig-minors file and the (gpu, gi, ci) tuple
	panic("Not implemented: GetMinorNumber")
}

func (c *MockA100ComputeInstance) SetMigMode(Mode int) (Return, Return) {
	panic("Not implemented: SetMigMode")
}

func (c *MockA100ComputeInstance) GetMigMode() (int, int, Return) {
	panic("Not implemented: GetMigMode")
}

func (c *MockA100ComputeInstance) GetGpuInstanceProfileInfo(Profile int) (GpuInstanceProfileInfo, Return) {
	panic("Not implemented: GetGpuInstanceProfileInfo")
}

func (c *MockA100ComputeInstance) CreateGpuInstance(Info *GpuInstanceProfileInfo) (GpuInstance, Return) {
	panic("Not implemented: CreateGpuInstance")
}

func (c *MockA100ComputeInstance) GetGpuInstances(Info *GpuInstanceProfileInfo) ([]GpuInstance, Return) {
	panic("Not implemented: GetGpuInstances")
}

func (c *MockA100ComputeInstance) GetMaxMigDeviceCount() (int, Return) {
	panic("Not implemented: GetMaxMigDeviceCount")
}

func (c *MockA100ComputeInstance) GetMigDeviceHandleByIndex(Index int) (Device, Return) {
	panic("Not implemented: GetMigDeviceHandleByIndex")
}

func (c *MockA100ComputeInstance) GetDeviceHandleFromMigDeviceHandle() (Device, Return) {
	return c.Info.Device, MockReturn(SUCCESS)
}

func (c *MockA100ComputeInstance) IsMigDeviceHandle() (bool, Return) {
	return true, MockReturn(SUCCESS)
}

func (c *MockA100ComputeInstance) GetComputeInstanceId() (int, Return) {
	return int(c.Info.Id), MockReturn(SUCCESS)
}

func (c *MockA100ComputeInstance) GetGPUInstanceId() (int, Return) {
	info, r := c.Info.GpuInstance.GetInfo()
	if r.Value() != SUCCESS {
		return 0, MockReturn(r.Value())
	}
	return int(info.Id), MockReturn(SUCCESS)
}
