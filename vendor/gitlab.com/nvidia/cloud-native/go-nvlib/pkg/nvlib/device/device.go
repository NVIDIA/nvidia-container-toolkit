/*
 * Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package device

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/dl"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

// Device defines the set of extended functions associated with a device.Device
type Device interface {
	nvml.Device
	GetMigDevices() ([]MigDevice, error)
	GetMigProfiles() ([]MigProfile, error)
	IsMigCapable() (bool, error)
	IsMigEnabled() (bool, error)
	VisitMigDevices(func(j int, m MigDevice) error) error
	VisitMigProfiles(func(p MigProfile) error) error
}

type device struct {
	nvml.Device
	lib *devicelib
}

var _ Device = &device{}

// NewDevice builds a new Device from an nvml.Device
func (d *devicelib) NewDevice(dev nvml.Device) (Device, error) {
	return &device{dev, d}, nil
}

// IsMigCapable checks if a device is capable of having MIG paprtitions created on it
func (d *device) IsMigCapable() (bool, error) {
	err := nvmlLookupSymbol("nvmlDeviceGetMigMode")
	if err != nil {
		return false, nil
	}

	_, _, ret := nvml.Device(d).GetMigMode()
	if ret == nvml.ERROR_NOT_SUPPORTED {
		return false, nil
	}
	if ret != nvml.SUCCESS {
		return false, fmt.Errorf("error getting MIG mode: %v", ret)
	}

	return true, nil
}

// IsMigEnabled checks if a device has MIG mode currently enabled on it
func (d *device) IsMigEnabled() (bool, error) {
	err := nvmlLookupSymbol("nvmlDeviceGetMigMode")
	if err != nil {
		return false, nil
	}

	mode, _, ret := nvml.Device(d).GetMigMode()
	if ret == nvml.ERROR_NOT_SUPPORTED {
		return false, nil
	}
	if ret != nvml.SUCCESS {
		return false, fmt.Errorf("error getting MIG mode: %v", ret)
	}

	return (mode == nvml.DEVICE_MIG_ENABLE), nil
}

// VisitMigDevices walks a top-level device and invokes a callback function for each MIG device configured on it
func (d *device) VisitMigDevices(visit func(int, MigDevice) error) error {
	count, ret := nvml.Device(d).GetMaxMigDeviceCount()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("error getting max MIG device count: %v", ret)
	}

	for i := 0; i < count; i++ {
		device, ret := nvml.Device(d).GetMigDeviceHandleByIndex(i)
		if ret == nvml.ERROR_NOT_FOUND {
			continue
		}
		if ret == nvml.ERROR_INVALID_ARGUMENT {
			continue
		}
		if ret != nvml.SUCCESS {
			return fmt.Errorf("error getting MIG device handle at index '%v': %v", i, ret)
		}
		mig, err := d.lib.NewMigDevice(device)
		if err != nil {
			return fmt.Errorf("error creating new MIG device wrapper: %v", err)
		}
		err = visit(i, mig)
		if err != nil {
			return fmt.Errorf("error visiting MIG device: %v", err)
		}
	}
	return nil
}

// VisitMigProfiles walks a top-level device and invokes a callback function for each unique MIG Profile that can be configured on it
func (d *device) VisitMigProfiles(visit func(MigProfile) error) error {
	capable, err := d.IsMigCapable()
	if err != nil {
		return fmt.Errorf("error checking if GPU is MIG capable: %v", err)
	}

	if !capable {
		return nil
	}

	memory, ret := d.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("error getting device memory info: %v", ret)
	}

	for i := 0; i < nvml.GPU_INSTANCE_PROFILE_COUNT; i++ {
		giProfileInfo, ret := d.GetGpuInstanceProfileInfo(i)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("error getting GPU Instance profile info: %v", ret)
		}

		for j := 0; j < nvml.COMPUTE_INSTANCE_PROFILE_COUNT; j++ {
			for k := 0; k < nvml.COMPUTE_INSTANCE_ENGINE_PROFILE_COUNT; k++ {
				p, err := d.lib.NewMigProfile(i, j, k, giProfileInfo.MemorySizeMB, memory.Total)
				if err != nil {
					return fmt.Errorf("error creating MIG profile: %v", err)
				}

				err = visit(p)
				if err != nil {
					return fmt.Errorf("error visiting MIG profile: %v", err)
				}
			}
		}
	}
	return nil
}

// GetMigDevices gets the set of MIG devices associated with a top-level device
func (d *device) GetMigDevices() ([]MigDevice, error) {
	var migs []MigDevice
	err := d.VisitMigDevices(func(j int, m MigDevice) error {
		migs = append(migs, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return migs, nil
}

// GetMigProfiles gets the set of unique MIG profiles associated with a top-level device
func (d *device) GetMigProfiles() ([]MigProfile, error) {
	var profiles []MigProfile
	err := d.VisitMigProfiles(func(p MigProfile) error {
		profiles = append(profiles, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

// VisitDevices visits each top-level device and invokes a callback function for it
func (d *devicelib) VisitDevices(visit func(int, Device) error) error {
	count, ret := d.nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("error getting device count: %v", ret)
	}

	for i := 0; i < count; i++ {
		device, ret := d.nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("error getting device handle for index '%v': %v", i, ret)
		}
		dev, err := d.NewDevice(device)
		if err != nil {
			return fmt.Errorf("error creating new device wrapper: %v", err)
		}
		err = visit(i, dev)
		if err != nil {
			return fmt.Errorf("error visiting device: %v", err)
		}
	}
	return nil
}

// VisitMigDevices walks a top-level device and invokes a callback function for each MIG device configured on it
func (d *devicelib) VisitMigDevices(visit func(int, Device, int, MigDevice) error) error {
	err := d.VisitDevices(func(i int, dev Device) error {
		err := dev.VisitMigDevices(func(j int, mig MigDevice) error {
			err := visit(i, dev, j, mig)
			if err != nil {
				return fmt.Errorf("error visiting MIG device: %v", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error visiting device: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error visiting devices: %v", err)
	}
	return nil
}

// VisitMigProfiles walks a top-level device and invokes a callback function for each unique MIG profile found on them
func (d *devicelib) VisitMigProfiles(visit func(MigProfile) error) error {
	visited := make(map[string]bool)
	err := d.VisitDevices(func(i int, dev Device) error {
		err := dev.VisitMigProfiles(func(p MigProfile) error {
			if visited[p.String()] {
				return nil
			}

			err := visit(p)
			if err != nil {
				return fmt.Errorf("error visiting MIG profile: %v", err)
			}

			visited[p.String()] = true
			return nil
		})
		if err != nil {
			return fmt.Errorf("error visiting device: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error visiting devices: %v", err)
	}
	return nil
}

// GetDevices gets the set of all top-level devices
func (d *devicelib) GetDevices() ([]Device, error) {
	var devs []Device
	err := d.VisitDevices(func(i int, dev Device) error {
		devs = append(devs, dev)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return devs, nil
}

// GetMigDevices gets the set of MIG devices across all top-level devices
func (d *devicelib) GetMigDevices() ([]MigDevice, error) {
	var migs []MigDevice
	err := d.VisitMigDevices(func(i int, dev Device, j int, m MigDevice) error {
		migs = append(migs, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return migs, nil
}

// GetMigProfiles gets the set of unique MIG profiles across all top-level devices
func (d *devicelib) GetMigProfiles() ([]MigProfile, error) {
	var profiles []MigProfile
	err := d.VisitMigProfiles(func(p MigProfile) error {
		profiles = append(profiles, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

// nvmlLookupSymbol checks to see if the given symbol is present in the NVML library
func nvmlLookupSymbol(symbol string) error {
	lib := dl.New("libnvidia-ml.so.1", dl.RTLD_LAZY|dl.RTLD_GLOBAL)
	if lib == nil {
		return fmt.Errorf("error instantiating DynamicLibrary for NVML")
	}
	err := lib.Open()
	if err != nil {
		return fmt.Errorf("error opening DynamicLibrary for NVML: %v", err)
	}
	defer lib.Close()
	return lib.Lookup(symbol)
}
