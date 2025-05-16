package discover

import (
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// NewCUDACompatHookDiscoverer creates a discoverer for a enable-cuda-compat hook.
// This hook is responsible for setting up CUDA compatibility in the container and depends on the host driver version.
func NewCUDACompatHookDiscoverer(logger logger.Interface, hookCreator HookCreator, driver *root.Driver) Discover {
	_, cudaVersionPattern := getCUDALibRootAndVersionPattern(logger, driver)
	var args []string
	if !strings.Contains(cudaVersionPattern, "*") {
		args = append(args, "--host-driver-version="+cudaVersionPattern)
	}

	return hookCreator.Create("enable-cuda-compat", args...)
}
