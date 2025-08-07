package discover

import (
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// NewCUDACompatHookDiscoverer creates a discoverer for a enable-cuda-compat hook.
// This hook is responsible for setting up CUDA compatibility in the container and depends on the host driver version.
func NewCUDACompatHookDiscoverer(logger logger.Interface, hookCreator HookCreator, version string) Discover {
	var args []string
	if version != "" && !strings.Contains(version, "*") {
		args = append(args, "--host-driver-version="+version)
	}

	return hookCreator.Create("enable-cuda-compat", args...)
}
