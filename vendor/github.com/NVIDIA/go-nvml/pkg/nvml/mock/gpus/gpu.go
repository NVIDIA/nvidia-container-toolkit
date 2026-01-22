package gpus

import "github.com/NVIDIA/go-nvml/pkg/nvml"

func Multiple(count int, gpu Config) []Config {
	gpus := make([]Config, count)
	for i := range gpus {
		gpus[i] = gpu
	}
	return gpus
}

// Config contains the minimal configuration needed for a GPU generation
type Config struct {
	Name         string
	Architecture nvml.DeviceArchitecture
	Brand        nvml.BrandType
	MemoryMB     uint64
	CudaMajor    int
	CudaMinor    int
	//Deprecated: Use PciInfo directly
	PciDeviceId uint32
	PciInfo     *nvml.PciInfo
	MIGProfiles MIGProfileConfig
}

// MIGProfileConfig contains MIG profile configuration for a GPU
type MIGProfileConfig struct {
	GpuInstanceProfiles       map[int]nvml.GpuInstanceProfileInfo
	ComputeInstanceProfiles   map[int]map[int]nvml.ComputeInstanceProfileInfo
	GpuInstancePlacements     map[int][]nvml.GpuInstancePlacement
	ComputeInstancePlacements map[int]map[int][]nvml.ComputeInstancePlacement
}
