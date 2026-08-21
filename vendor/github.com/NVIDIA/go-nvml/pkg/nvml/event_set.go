// Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nvml

import "unsafe"

// EventData includes an interface type for Device instead of nvmlDevice
type EventData struct {
	Device            Device
	EventType         uint64
	EventData         uint64
	GpuInstanceId     uint32
	ComputeInstanceId uint32
}

func (e nvmlEventData) convert() EventData {
	out := EventData{
		Device:            e.Device,
		EventType:         e.EventType,
		EventData:         e.EventData,
		GpuInstanceId:     e.GpuInstanceId,
		ComputeInstanceId: e.ComputeInstanceId,
	}
	return out
}

// nvml.EventSetCreate()
func (l *library) EventSetCreate() (EventSet, Return) {
	var Set nvmlEventSet
	ret := nvmlEventSetCreate(&Set)
	return Set, ret
}

// nvml.EventSetWait()
func (l *library) EventSetWait(set EventSet, timeoutms uint32) (EventData, Return) {
	return set.Wait(timeoutms)
}

func (set nvmlEventSet) Wait(timeoutms uint32) (EventData, Return) {
	var data nvmlEventData
	ret := nvmlEventSetWait(set, &data, timeoutms)
	return data.convert(), ret
}

// nvml.EventSetFree()
func (l *library) EventSetFree(set EventSet) Return {
	return set.Free()
}

func (set nvmlEventSet) Free() Return {
	return nvmlEventSetFree(set)
}

// nvml.SystemEventSetCreate()
func (l *library) SystemEventSetCreate(request *SystemEventSetCreateRequest) Return {
	return nvmlSystemEventSetCreate(request)
}

// nvml.SystemEventSetFree()
func (l *library) SystemEventSetFree(request *SystemEventSetFreeRequest) Return {
	return nvmlSystemEventSetFree(request)
}

// nvml.SystemRegisterEvents()
func (l *library) SystemRegisterEvents(request *SystemRegisterEventRequest) Return {
	return nvmlSystemRegisterEvents(request)
}

// nvml.SystemEventSetWait()
func (l *library) SystemEventSetWait(request *SystemEventSetWaitRequest) Return {
	return nvmlSystemEventSetWait(request)
}

// nvml.EventSetRegisterGpuOperationalEvents_v1()
func (l *library) EventSetRegisterGpuOperationalEvents_v1(set EventSet, config *GpuOperationalEventConfig_v1) Return {
	return set.RegisterGpuOperationalEvents_v1(config)
}

func (set nvmlEventSet) RegisterGpuOperationalEvents_v1(config *GpuOperationalEventConfig_v1) Return {
	return nvmlEventSetRegisterGpuOperationalEvents_v1(set, config)
}

// nvml.EventSetWait_v3()
func (l *library) EventSetWait_v3(set EventSet, timeoutms uint32) (EventData_v2, Return) {
	return set.Wait_v3(timeoutms)
}

func (set nvmlEventSet) Wait_v3(timeoutms uint32) (EventData_v2, Return) {
	var data EventData_v2
	ret := nvmlEventSetWait_v3(set, &data, timeoutms)
	return data, ret
}

// nvml.EventSetGetContextCount_v1()
func (l *library) EventSetGetContextCount_v1(set EventSet) (uint32, Return) {
	return set.GetContextCount_v1()
}

func (set nvmlEventSet) GetContextCount_v1() (uint32, Return) {
	var count uint32
	ret := nvmlEventSetGetContextCount_v1(set, &count)
	return count, ret
}

// nvml.EventSetGetContextInfo_v1()
func (l *library) EventSetGetContextInfo_v1(set EventSet, index uint32) (OperationalEventContextInfo_v1, Return) {
	return set.GetContextInfo_v1(index)
}

func (set nvmlEventSet) GetContextInfo_v1(index uint32) (OperationalEventContextInfo_v1, Return) {
	var info OperationalEventContextInfo_v1
	ret := nvmlEventSetGetContextInfo_v1(set, index, &info)
	return info, ret
}

// nvml.EventSetGetContextData_v1()
func (l *library) EventSetGetContextData_v1(set EventSet, index uint32, data []byte) (uint32, Return) {
	return set.GetContextData_v1(index, data)
}

func (set nvmlEventSet) GetContextData_v1(index uint32, data []byte) (uint32, Return) {
	dataSize := uint32(len(data))
	var ptr unsafe.Pointer
	if len(data) > 0 {
		ptr = unsafe.Pointer(&data[0])
	}
	ret := nvmlEventSetGetContextData_v1(set, index, ptr, &dataSize)
	return dataSize, ret
}

// nvml.EventSetGetGpuOperationalEventContextLegacyXid_v1()
func (l *library) EventSetGetGpuOperationalEventContextLegacyXid_v1(set EventSet, index uint32) (GpuOperationalEventContextLegacyXid_v1, Return) {
	return set.GetGpuOperationalEventContextLegacyXid_v1(index)
}

func (set nvmlEventSet) GetGpuOperationalEventContextLegacyXid_v1(index uint32) (GpuOperationalEventContextLegacyXid_v1, Return) {
	var xid GpuOperationalEventContextLegacyXid_v1
	ret := nvmlEventSetGetGpuOperationalEventContextLegacyXid_v1(set, index, &xid)
	return xid, ret
}
