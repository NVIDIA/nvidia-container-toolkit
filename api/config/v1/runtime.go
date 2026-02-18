/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package config

import "github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"

// RuntimeConfig stores the config options for the NVIDIA Container Runtime
type RuntimeConfig struct {
	DebugFilePath string `toml:"debug"`
	// LogLevel defines the logging level for the application
	LogLevel string `toml:"log-level"`
	// Runtimes defines the candidates for the low-level runtime
	Runtimes []string    `toml:"runtimes"`
	Mode     string      `toml:"mode"`
	Modes    modesConfig `toml:"modes"`
}

// modesConfig defines (optional) per-mode configs
type modesConfig struct {
	CSV    csvModeConfig    `toml:"csv"`
	CDI    cdiModeConfig    `toml:"cdi"`
	JitCDI jitCDIModeConfig `toml:"jit-cdi,omitempty"`
	Legacy legacyModeConfig `toml:"legacy"`
}

type cdiModeConfig struct {
	// SpecDirs allows for the default spec dirs for CDI to be overridden
	SpecDirs []string `toml:"spec-dirs"`
	// DefaultKind sets the default kind to be used when constructing fully-qualified CDI device names
	DefaultKind string `toml:"default-kind"`
	// AnnotationPrefixes sets the allowed prefixes for CDI annotation-based device injection
	AnnotationPrefixes []string `toml:"annotation-prefixes"`
}

type jitCDIModeConfig struct {
	// NVCDIFeatureFlags sets a list of nvcdi features explicitly.
	NVCDIFeatureFlags []nvcdi.FeatureFlag `toml:"nvcdi-feature-flags,omitempty"`
}

type csvModeConfig struct {
	MountSpecPath string `toml:"mount-spec-path"`
	// CompatContainerRoot specifies the compat root used when the the standard
	// CUDA compat libraries should not be used.
	CompatContainerRoot string `toml:"compat-container-root,omitempty"`
}

type legacyModeConfig struct {
	// CUDACompatMode sets the mode to be used to make CUDA Forward Compat
	// libraries discoverable in the container.
	CUDACompatMode cudaCompatMode `toml:"cuda-compat-mode,omitempty"`
}

type cudaCompatMode string

const (
	defaultCUDACompatMode = CUDACompatModeLdconfig
	// CUDACompatModeDisabled explicitly disables the handling of CUDA Forward
	// Compatibility in the NVIDIA Container Runtime and NVIDIA Container
	// Runtime Hook.
	CUDACompatModeDisabled = cudaCompatMode("disabled")
	// CUDACompatModeHook uses a container lifecycle hook to implement CUDA
	// Forward Compatibility support. This requires the use of the NVIDIA
	// Container Runtime and is not compatible with use cases where only the
	// NVIDIA Container Runtime Hook is used (e.g. the Docker --gpus flag).
	CUDACompatModeHook = cudaCompatMode("hook")
	// CUDACompatModeLdconfig adds the folders containing CUDA Forward Compat
	// libraries to the ldconfig command invoked from the NVIDIA Container
	// Runtime Hook.
	CUDACompatModeLdconfig = cudaCompatMode("ldconfig")
	// CUDACompatModeMount mounts CUDA Forward Compat folders from the container
	// to the container when using the NVIDIA Container Runtime Hook.
	CUDACompatModeMount = cudaCompatMode("mount")
)
