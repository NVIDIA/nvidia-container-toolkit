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

package operator

import "path/filepath"

const (
	defaultRuntimeName = "nvidia"

	defaultRoot = "/usr/bin"
)

// Runtime defines a runtime to be configured.
// The path and whether the runtime is the default runtime can be specfied
type Runtime struct {
	name         string
	Path         string
	SetAsDefault bool
}

// Runtimes defines a set of runtimes to be configure for use in the GPU Operator
type Runtimes map[string]Runtime

type config struct {
	root              string
	nvidiaRuntimeName string
	setAsDefault      bool
}

// GetRuntimes returns the set of runtimes to be configured for use with the GPU Operator.
func GetRuntimes(opts ...Option) Runtimes {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}

	if c.root == "" {
		c.root = defaultRoot
	}
	if c.nvidiaRuntimeName == "" {
		c.nvidiaRuntimeName = defaultRuntimeName
	}

	runtimes := make(Runtimes)
	runtimes.add(c.nvidiaRuntime())

	modes := []string{"cdi", "legacy"}
	for _, mode := range modes {
		runtimes.add(c.modeRuntime(mode))
	}
	return runtimes
}

// DefaultRuntimeName returns the name of the default runtime.
func (r Runtimes) DefaultRuntimeName() string {
	for _, runtime := range r {
		if runtime.SetAsDefault {
			return runtime.name
		}
	}
	return ""
}

// Add a runtime to the set of runtimes.
func (r *Runtimes) add(runtime Runtime) {
	(*r)[runtime.name] = runtime
}

// nvidiaRuntime creates a runtime that corresponds to the nvidia runtime.
// If name is equal to one of the predefined runtimes, `nvidia` is used as the runtime name instead.
func (c config) nvidiaRuntime() Runtime {
	predefinedRuntimes := map[string]struct{}{
		"nvidia-cdi":    {},
		"nvidia-legacy": {},
	}
	name := c.nvidiaRuntimeName
	if _, isPredefinedRuntime := predefinedRuntimes[name]; isPredefinedRuntime {
		name = defaultRuntimeName
	}
	return c.newRuntime(name, "nvidia-container-runtime")
}

// modeRuntime creates a runtime for the specified mode.
func (c config) modeRuntime(mode string) Runtime {
	return c.newRuntime("nvidia-"+mode, "nvidia-container-runtime."+mode)
}

// newRuntime creates a runtime based on the configuration
func (c config) newRuntime(name string, binary string) Runtime {
	return Runtime{
		name:         name,
		Path:         filepath.Join(c.root, binary),
		SetAsDefault: c.setAsDefault && name == c.nvidiaRuntimeName,
	}
}

// Option is a functional option for configuring set of runtimes.
type Option func(*config)

// WithRoot sets the root directory for the runtime binaries.
func WithRoot(root string) Option {
	return func(c *config) {
		c.root = root
	}
}

// WithNvidiaRuntimeName sets the name of the nvidia runtime.
func WithNvidiaRuntimeName(name string) Option {
	return func(c *config) {
		c.nvidiaRuntimeName = name
	}
}

// WithSetAsDefault sets the default runtime to the nvidia runtime.
func WithSetAsDefault(set bool) Option {
	return func(c *config) {
		c.setAsDefault = set
	}
}
