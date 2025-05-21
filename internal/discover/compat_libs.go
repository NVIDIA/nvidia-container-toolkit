package discover

import (
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/hooks"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// cudaCompatHook is a discoverer for the enable-cuda-compat hook.
type cudaCompatHook struct {
	hooks.Hook
}

// NewCUDACompatHookDiscoverer creates a discoverer for a enable-cuda-compat hook.
// This hook is responsible for setting up CUDA compatibility in the container and depends on the host driver version.
func NewCUDACompatHookDiscoverer(logger logger.Interface, hookCreator hooks.HookCreator, driver *root.Driver) Discover {
	_, cudaVersionPattern := getCUDALibRootAndVersionPattern(logger, driver)
	var args []string
	if !strings.Contains(cudaVersionPattern, "*") {
		args = append(args, "--host-driver-version="+cudaVersionPattern)
	}

	hook := hookCreator.Create(hooks.EnableCudaCompat, args...)
	if hook == nil {
		return nil
	}

	return &cudaCompatHook{
		Hook: *hook,
	}
}

func (h *cudaCompatHook) Hooks() ([]Hook, error) {
	return []Hook{
		{
			Lifecycle: h.Lifecycle,
			Path:      h.Path,
			Args:      h.Args,
		}}, nil
}

func (h *cudaCompatHook) Devices() ([]Device, error) {
	return nil, nil
}

func (h *cudaCompatHook) Mounts() ([]Mount, error) {
	return nil, nil
}
