package nvcdi

import (
	"fmt"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type cudaMemoryLimits struct {
	logger      logger.Interface
	driverRoot  string
	uuid        string
	hookCreator discover.HookCreator
}

func (l *nvcdilib) newCudaMemoryLimits(d device.Device) (discover.Discover, error) {
	uuid, nvmlRet := d.GetUUID()
	if nvmlRet != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device UUID: %w", nvmlRet)
	}

	cMemLimits := &cudaMemoryLimits{
		logger:      l.logger,
		driverRoot:  l.driver.Root,
		uuid:        uuid,
		hookCreator: l.hookCreator,
	}

	return cMemLimits, nil
}

// Devices are empty for this discoverer
func (c *cudaMemoryLimits) Devices() ([]discover.Device, error) {
	return nil, nil
}

// EnvVars are empty for this discoverer
func (c *cudaMemoryLimits) EnvVars() ([]discover.EnvVar, error) {
	return nil, nil
}

// Hooks returns a set of hooks that assigns a CUDA memory limit to the cgroup of the GPU workload container
func (c *cudaMemoryLimits) Hooks() ([]discover.Hook, error) {
	return c.hookCreator.Create("apply-cuda-memory-limits", c.driverRoot, c.uuid).Hooks()
}

// Mounts are empty for this discoverer
func (c *cudaMemoryLimits) Mounts() ([]discover.Mount, error) {
	return nil, nil
}
