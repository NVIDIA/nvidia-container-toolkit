package discover

import (
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// EnableCUDACompatHookOptions defines the options that can be specified
// when creating the enable-cuda-compat hook.
type EnableCUDACompatHookOptions struct {
	HostDriverVersion       string
	CUDACompatContainerRoot string
}

// NewCUDACompatHookDiscoverer creates a discoverer for a enable-cuda-compat hook.
// This hook is responsible for setting up CUDA compatibility in the container and depends on the host driver version.
func NewCUDACompatHookDiscoverer(logger logger.Interface, hookCreator HookCreator, o *EnableCUDACompatHookOptions) Discover {
	return hookCreator.Create(EnableCudaCompatHook, o.args()...)
}

func (o *EnableCUDACompatHookOptions) args() []string {
	if o == nil {
		return nil
	}
	var args []string
	if o.HostDriverVersion != "" && !strings.Contains(o.HostDriverVersion, "*") {
		args = append(args, "--host-driver-version="+o.HostDriverVersion)
	}
	if o.CUDACompatContainerRoot != "" {
		args = append(args, "--cuda-compat-container-root="+o.CUDACompatContainerRoot)
	}
	return args
}
