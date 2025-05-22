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

package discover

import (
	"path/filepath"

	"tags.cncf.io/container-device-interface/pkg/cdi"
)

var _ Discover = (*Hook)(nil)

// Devices returns an empty list of devices for a Hook discoverer.
func (h *Hook) Devices() ([]Device, error) {
	return nil, nil
}

// Mounts returns an empty list of mounts for a Hook discoverer.
func (h *Hook) Mounts() ([]Mount, error) {
	return nil, nil
}

// Hooks allows the Hook type to also implement the Discoverer interface.
// It returns a single hook
func (h *Hook) Hooks() ([]Hook, error) {
	if h == nil {
		return nil, nil
	}

	return []Hook{*h}, nil
}

type HookName string

// disabledHooks allows individual hooks to be disabled.
type disabledHooks map[HookName]bool

const (
	// HookEnableCudaCompat refers to the hook used to enable CUDA Forward Compatibility.
	// This was added with v1.17.5 of the NVIDIA Container Toolkit.
	HookEnableCudaCompat = HookName("enable-cuda-compat")
	// directory path to be mounted into a container.
	HookCreateSymlinks = HookName("create-symlinks")
	// HookUpdateLDCache refers to the hook used to  Update the dynamic linker
	// cache inside the directory path to be mounted into a container.
	HookUpdateLDCache = HookName("update-ldcache")
)

// AllHooks maintains a future-proof list of all defined hooks.
var AllHooks = []HookName{
	HookEnableCudaCompat,
	HookCreateSymlinks,
	HookUpdateLDCache,
}

type Option func(*CDIHook)

type CDIHook struct {
	nvidiaCDIHookPath string
	disabledHooks     disabledHooks
}

type HookCreator interface {
	Create(HookName, ...string) *Hook
}

func WithDisabledHooks(hooks ...HookName) Option {
	return func(c *CDIHook) {
		for _, hook := range hooks {
			c.disabledHooks[hook] = true
		}
	}
}

func NewHookCreator(nvidiaCDIHookPath string, opts ...Option) HookCreator {
	CDIHook := &CDIHook{
		nvidiaCDIHookPath: nvidiaCDIHookPath,
		disabledHooks:     disabledHooks{},
	}

	for _, opt := range opts {
		opt(CDIHook)
	}

	return CDIHook
}

func (c CDIHook) Create(name HookName, args ...string) *Hook {
	if c.disabledHooks[name] {
		return nil
	}

	if name == "create-symlinks" {
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
		Args:      append(c.requiredArgs(string(name)), args...),
	}
}

func (c CDIHook) requiredArgs(name string) []string {
	base := filepath.Base(c.nvidiaCDIHookPath)
	if base == "nvidia-ctk" {
		return []string{base, "hook", name}
	}
	return []string{base, name}
}
