/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package hooks

import (
	"path/filepath"

	"tags.cncf.io/container-device-interface/pkg/cdi"
)

// A HookName refers to one of the predefined set of CDI hooks that may be
// included in the generated CDI specification.
type HookName string

// DisabledHooks allows individual hooks to be disabled.
type DisabledHooks map[HookName]bool

const (
	// EnableCudaCompat refers to the hook used to enable CUDA Forward Compatibility.
	// This was added with v1.17.5 of the NVIDIA Container Toolkit.
	EnableCudaCompat = HookName("enable-cuda-compat")
	// CreateSymlinks refers to the hook used to create symlinks for the NVIDIA
	// Container Toolkit. This was added with v1.17.5 of the NVIDIA Container Toolkit.
	CreateSymlinks = HookName("create-symlinks")
	// UpdateLDCache refers to the hook used to update the LD cache for the NVIDIA
	// Container Toolkit. This was added with v1.17.5 of the NVIDIA Container Toolkit.
	UpdateLDCache = HookName("update-ldcache")
)

// Hook represents an OCI container hook.
type Hook struct {
	Lifecycle string
	Path      string
	Args      []string
}

// Option is a function that configures the nvcdilib
type Option func(*CDIHook)

type CDIHook struct {
	nvidiaCDIHookPath string
	disabledHooks     DisabledHooks
}

type HookCreator interface {
	Create(HookName, ...string) *Hook
}

func NewHookCreator(nvidiaCDIHookPath string, disabledHooks DisabledHooks) HookCreator {
	if disabledHooks == nil {
		disabledHooks = make(DisabledHooks)
	}

	CDIHook := &CDIHook{
		nvidiaCDIHookPath: nvidiaCDIHookPath,
		disabledHooks:     disabledHooks,
	}

	return CDIHook
}

func (c CDIHook) Create(name HookName, args ...string) *Hook {
	if c.disabledHooks[name] {
		return nil
	}

	if name == CreateSymlinks {
		if len(args) == 0 {
			return nil
		}

		links := []string{}
		for _, arg := range args {
			links = append(links, "--link", arg)
		}
		args = links
	}

	return &Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      c.nvidiaCDIHookPath,
		Args:      append(c.requiredArgs(name), args...),
	}
}

func (c CDIHook) requiredArgs(name HookName) []string {
	base := filepath.Base(c.nvidiaCDIHookPath)
	if base == "nvidia-ctk" {
		return []string{base, "hook", string(name)}
	}
	return []string{base, string(name)}
}

// IsSupported checks whether a hook of the specified name is supported.
// Hooks must be explicitly disabled, meaning that if no disabled hooks are
// all hooks are supported.
func (c CDIHook) IsSupported(h HookName) bool {
	if len(c.disabledHooks) == 0 {
		return true
	}
	return !c.disabledHooks[h]
}
