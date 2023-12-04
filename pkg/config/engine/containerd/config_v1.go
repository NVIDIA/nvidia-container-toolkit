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

package containerd

import (
	"fmt"

	"github.com/pelletier/go-toml"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
)

// ConfigV1 represents a version 1 containerd config
type ConfigV1 Config

var _ engine.Interface = (*ConfigV1)(nil)

// AddRuntime adds a runtime to the containerd config
func (c *ConfigV1) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil || c.Tree == nil {
		return fmt.Errorf("config is nil")
	}

	config := *c.Tree

	config.Set("version", int64(1))

	if runc, ok := config.GetPath([]string{"plugins", "cri", "containerd", "runtimes", "runc"}).(*toml.Tree); ok {
		runc, _ = toml.Load(runc.String())
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name}, runc)
	}

	if config.GetPath([]string{"plugins", "cri", "containerd", "runtimes", name}) == nil {
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "runtime_type"}, c.RuntimeType)
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "runtime_root"}, "")
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "runtime_engine"}, "")
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "privileged_without_host_devices"}, false)
	}

	if len(c.ContainerAnnotations) > 0 {
		annotations, err := (*Config)(c).getRuntimeAnnotations([]string{"plugins", "cri", "containerd", "runtimes", name, "container_annotations"})
		if err != nil {
			return err
		}
		annotations = append(c.ContainerAnnotations, annotations...)
		config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "container_annotations"}, annotations)
	}

	config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "options", "BinaryName"}, path)
	config.SetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "options", "Runtime"}, path)

	if setAsDefault && c.UseDefaultRuntimeName {
		config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime_name"}, name)
	} else if setAsDefault {
		// Note: This is deprecated in containerd 1.4.0 and will be removed in 1.5.0
		if config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime"}) == nil {
			config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "runtime_type"}, c.RuntimeType)
			config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "runtime_root"}, "")
			config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "runtime_engine"}, "")
			config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "privileged_without_host_devices"}, false)
		}
		config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "options", "BinaryName"}, path)
		config.SetPath([]string{"plugins", "cri", "containerd", "default_runtime", "options", "Runtime"}, path)
	}

	*c.Tree = config
	return nil
}

// DefaultRuntime returns the default runtime for the cri-o config
func (c ConfigV1) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", "cri", "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the docker config
func (c *ConfigV1) RemoveRuntime(name string) error {
	if c == nil || c.Tree == nil {
		return nil
	}

	config := *c.Tree

	// If the specified runtime was set as the default runtime we need to remove the default runtime too.
	runtimePath, ok := config.GetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "options", "BinaryName"}).(string)
	if !ok || runtimePath == "" {
		runtimePath, _ = config.GetPath([]string{"plugins", "cri", "containerd", "runtimes", name, "options", "Runtime"}).(string)
	}
	defaultRuntimePath, ok := config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime", "options", "BinaryName"}).(string)
	if !ok || defaultRuntimePath == "" {
		defaultRuntimePath, _ = config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime", "options", "Runtime"}).(string)
	}
	if runtimePath != "" && defaultRuntimePath != "" && runtimePath == defaultRuntimePath {
		config.DeletePath([]string{"plugins", "cri", "containerd", "default_runtime"})
	}

	config.DeletePath([]string{"plugins", "cri", "containerd", "runtimes", name})
	if runtime, ok := config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime_name"}).(string); ok {
		if runtime == name {
			config.DeletePath([]string{"plugins", "cri", "containerd", "default_runtime_name"})
		}
	}

	runtimeConfigPath := []string{"plugins", "cri", "containerd", "runtimes", name}
	for i := 0; i < len(runtimeConfigPath); i++ {
		if runtimes, ok := config.GetPath(runtimeConfigPath[:len(runtimeConfigPath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				config.DeletePath(runtimeConfigPath[:len(runtimeConfigPath)-i])
			}
		}
	}

	if len(config.Keys()) == 1 && config.Keys()[0] == "version" {
		config.Delete("version")
	}

	*c.Tree = config
	return nil
}

// SetOption sets the specified containerd option.
func (c *ConfigV1) Set(key string, value interface{}) {
	config := *c.Tree
	config.SetPath([]string{"plugins", "cri", "containerd", key}, value)
	*c.Tree = config
}

// Save wrotes the config to a file
func (c ConfigV1) Save(path string) (int64, error) {
	return (Config)(c).Save(path)
}
