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

package crio

import (
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/engine"
	"github.com/pelletier/go-toml"
)

// Config represents the cri-o config
type Config toml.Tree

// New creates a cri-o config with the specified options
func New(opts ...Option) (engine.Interface, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b.build()
}

// AddRuntime adds a new runtime to the crio config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	config := (toml.Tree)(*c)

	switch runc := config.Get("crio.runtime.runtimes.runc").(type) {
	case *toml.Tree:
		runc, _ = toml.Load(runc.String())
		config.SetPath([]string{"crio", "runtime", "runtimes", name}, runc)
	}

	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_path"}, path)
	config.SetPath([]string{"crio", "runtime", "runtimes", name, "runtime_type"}, "oci")

	if setAsDefault {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, name)
	}

	*c = (Config)(config)
	return nil
}

// DefaultRuntime returns the default runtime for the cri-o config
func (c Config) DefaultRuntime() string {
	config := (toml.Tree)(c)
	if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the cri-o config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil {
		return nil
	}

	config := (toml.Tree)(*c)
	if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		if runtime == name {
			config.DeletePath([]string{"crio", "runtime", "default_runtime"})
		}
	}

	runtimeClassPath := []string{"crio", "runtime", "runtimes", name}
	config.DeletePath(runtimeClassPath)
	for i := 0; i < len(runtimeClassPath); i++ {
		remainingPath := runtimeClassPath[:len(runtimeClassPath)-i]
		if entry, ok := config.GetPath(remainingPath).(*toml.Tree); ok {
			if len(entry.Keys()) != 0 {
				break
			}
			config.DeletePath(remainingPath)
		}
	}

	*c = (Config)(config)
	return nil
}

// Save writes the config to the specified path
func (c Config) Save(path string) (int64, error) {
	config := (toml.Tree)(c)
	output, err := config.ToTomlString()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
	}

	if path == "" {
		os.Stdout.WriteString(fmt.Sprintf("%s\n", output))
		return int64(len(output)), nil
	}

	if len(output) == 0 {
		err := os.Remove(path)
		if err != nil {
			return 0, fmt.Errorf("unable to remove empty file: %v", err)
		}
		return 0, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("unable to open '%v' for writing: %v", path, err)
	}
	defer f.Close()

	n, err := f.WriteString(output)
	if err != nil {
		return 0, fmt.Errorf("unable to write output: %v", err)
	}

	return int64(n), err
}
