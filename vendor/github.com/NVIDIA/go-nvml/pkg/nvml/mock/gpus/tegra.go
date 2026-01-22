package gpus

import "github.com/NVIDIA/go-nvml/pkg/nvml"

var (
	THOR_IGX = Config{
		Name:         "NVIDIA Thor",
		Architecture: nvml.DEVICE_ARCH_AMPERE,
		Brand:        nvml.BRAND_NVIDIA,
		MemoryMB:     131882934272 / 1024 / 1024,
		CudaMajor:    11,
		CudaMinor:    0,
		PciInfo: &nvml.PciInfo{
			Domain: 0,
			Bus:    1,
			Device: 0,
		},
	}
)
