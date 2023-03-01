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
	"os"

	"github.com/pelletier/go-toml"
)

// AddRuntime adds a runtime to the containerd config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil || c.Tree == nil {
		return fmt.Errorf("config is nil")
	}
	config := *c.Tree

	config.Set("version", int64(2))

	switch runc := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "runc"}).(type) {
	case *toml.Tree:
		runc, _ = toml.Load(runc.String())
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}, runc)
	}

	if config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}) == nil {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "runtime_type"}, c.RuntimeType)
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "runtime_root"}, "")
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "runtime_engine"}, "")
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "privileged_without_host_devices"}, false)
	}

	cdiAnnotations := []interface{}{"cdi.k8s.io/*"}
	containerAnnotations, ok := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "container_annotations"}).([]interface{})
	if ok && containerAnnotations != nil {
		cdiAnnotations = append(containerAnnotations, cdiAnnotations...)
	}
	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "container_annotations"}, cdiAnnotations)

	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "options", "BinaryName"}, path)

	if setAsDefault {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}, name)
	}

	*c.Tree = config
	return nil
}

// DefaultRuntime returns the default runtime for the cri-o config
func (c Config) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the docker config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil || c.Tree == nil {
		return nil
	}

	config := *c.Tree

	config.DeletePath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name})
	if runtime, ok := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}).(string); ok {
		if runtime == name {
			config.DeletePath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
		}
	}

	runtimePath := []string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}
	for i := 0; i < len(runtimePath); i++ {
		if runtimes, ok := config.GetPath(runtimePath[:len(runtimePath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				config.DeletePath(runtimePath[:len(runtimePath)-i])
			}
		}
	}

	if len(config.Keys()) == 1 && config.Keys()[0] == "version" {
		config.Delete("version")
	}

	*c.Tree = config
	return nil
}

// Save writes the config to the specified path
func (c Config) Save(path string) (int64, error) {
	config := c.Tree
	output, err := config.ToTomlString()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
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
