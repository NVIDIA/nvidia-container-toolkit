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
	"fmt"
	"path/filepath"

	"tags.cncf.io/container-device-interface/pkg/cdi"
)

// A HookName represents a supported CDI hooks.
type HookName string

const (
	// AllHooks is a special hook name that allows all hooks to be matched.
	AllHooks = HookName("all")

	// A ChmodHook is used to set the file mode of the specified paths.
	//
	// Deprecated: The chmod hook is deprecated and will be removed in a future release.
	ChmodHook = HookName("chmod")
	// A CreateSymlinksHook is used to create symlinks in the container.
	CreateSymlinksHook = HookName("create-symlinks")
	// DisableDeviceNodeModificationHook refers to the hook used to ensure that
	// device nodes are not created by libnvidia-ml.so or nvidia-smi in a
	// container.
	// Added in v1.17.8
	DisableDeviceNodeModificationHook = HookName("disable-device-node-modification")
	// An EnableCudaCompatHook is used to enabled CUDA Forward Compatibility.
	// Added in v1.17.5
	EnableCudaCompatHook = HookName("enable-cuda-compat")
	// An UpdateLDCacheHook is the hook used to update the ldcache in the
	// container. This allows injected libraries to be discoverable.
	UpdateLDCacheHook = HookName("update-ldcache")

	defaultNvidiaCDIHookPath = "/usr/bin/nvidia-cdi-hook"
)

// defaultDisabledHooks defines hooks that are disabled by default.
// These hooks can be explicitly enabled using the WithEnabledHooks option.
var defaultDisabledHooks = []HookName{
	// ChmodHook is disabled by default as it was a workaround for older
	// versions of crun that has since been fixed.
	ChmodHook,
}

var _ Discover = (*Hook)(nil)

// Devices returns an empty list of devices for a Hook discoverer.
func (h *Hook) Devices() ([]Device, error) {
	return nil, nil
}

// EnvVars returns an empty list of envs for a Hook discoverer.
func (h *Hook) EnvVars() ([]EnvVar, error) {
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

type hookCreatorOptions struct {
	nvidiaCDIHookPath string
	disabledHooks     []HookName
	enabledHooks      []HookName
	debugLogging      bool
}

type Option func(*hookCreatorOptions)

type cdiHookCreator struct {
	nvidiaCDIHookPath string
	disabledHooks     map[HookName]bool

	fixedArgs    []string
	debugLogging bool
}

// An allDisabledHookCreator is a HookCreator that does not create any hooks.
type allDisabledHookCreator struct{}

// Create returns nil for all hooks for an allDisabledHookCreator.
func (a *allDisabledHookCreator) Create(name HookName, args ...string) *Hook {
	return nil
}

// A HookCreator defines an interface for creating discover hooks.
type HookCreator interface {
	Create(HookName, ...string) *Hook
}

func WithDebugLogging(debugLogging bool) Option {
	return func(hco *hookCreatorOptions) {
		hco.debugLogging = debugLogging
	}
}

// WithDisabledHooks explicitly disables the specified hooks.
// This can be specified multiple times.
func WithDisabledHooks(hooks ...HookName) Option {
	return func(c *hookCreatorOptions) {
		c.disabledHooks = append(c.disabledHooks, hooks...)
	}
}

// WithEnabledHooks explicitly enables the specified hooks.
// This is useful for enabling hooks that are disabled by default.
func WithEnabledHooks(hooks ...HookName) Option {
	return func(c *hookCreatorOptions) {
		c.enabledHooks = append(c.enabledHooks, hooks...)
	}
}

// WithNVIDIACDIHookPath sets the path to the nvidia-cdi-hook binary.
func WithNVIDIACDIHookPath(nvidiaCDIHookPath string) Option {
	return func(c *hookCreatorOptions) {
		c.nvidiaCDIHookPath = nvidiaCDIHookPath
	}
}

func NewHookCreator(opts ...Option) HookCreator {
	o := &hookCreatorOptions{
		nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
	}
	for _, opt := range opts {
		opt(o)
	}

	o.disabledHooks = append(o.disabledHooks, defaultDisabledHooks...)

	disabledHooks := make(map[HookName]bool)
	for _, h := range o.disabledHooks {
		disabledHooks[h] = true
	}

	if disabledHooks[AllHooks] && len(o.enabledHooks) == 0 {
		return &allDisabledHookCreator{}
	}

	for _, h := range o.enabledHooks {
		disabledHooks[h] = false
	}

	c := &cdiHookCreator{
		nvidiaCDIHookPath: o.nvidiaCDIHookPath,
		disabledHooks:     disabledHooks,
		fixedArgs:         getFixedArgsForCDIHookCLI(o.nvidiaCDIHookPath),
		debugLogging:      o.debugLogging,
	}

	return c
}

// Create creates a new hook with the given name and arguments.
// If a hook is disabled, a nil hook is returned.
func (c cdiHookCreator) Create(name HookName, args ...string) *Hook {
	if c.isDisabled(name, args...) {
		return nil
	}

	return &Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      c.nvidiaCDIHookPath,
		Args:      append(c.requiredArgs(name), c.transformArgs(name, args...)...),
		Env:       []string{fmt.Sprintf("NVIDIA_CTK_DEBUG=%v", c.debugLogging)},
	}
}

func (c cdiHookCreator) isDisabled(name HookName, args ...string) bool {
	disabled, ok := c.disabledHooks[name]
	if ok {
		return disabled
	}
	if c.disabledHooks[AllHooks] {
		return true
	}

	// still reject hooks that require args if none were provided
	switch name {
	case CreateSymlinksHook, ChmodHook:
		return len(args) == 0
	}
	return false
}

func (c cdiHookCreator) requiredArgs(name HookName) []string {
	return append(c.fixedArgs, string(name))
}

func (c cdiHookCreator) transformArgs(name HookName, args ...string) []string {
	switch name {
	case CreateSymlinksHook:
		var transformedArgs []string
		for _, arg := range args {
			transformedArgs = append(transformedArgs, "--link", arg)
		}
		return transformedArgs
	case ChmodHook:
		var transformedArgs = []string{"--mode", "755"}
		for _, arg := range args {
			transformedArgs = append(transformedArgs, "--path", arg)
		}
		return transformedArgs
	default:
		return args
	}
}

// getFixedArgsForCDIHookCLI returns the fixed arguments for the hook CLI.
// If the nvidia-ctk binary is used, hooks are implemented under the hook
// subcommand.
// For the nvidia-cdi-hook binary, the hooks are implemented as subcommands of
// the top-level CLI.
func getFixedArgsForCDIHookCLI(nvidiaCDIHookPath string) []string {
	base := filepath.Base(nvidiaCDIHookPath)
	if base == "nvidia-ctk" {
		return []string{base, "hook"}
	}
	return []string{base}
}
