/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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
*/

package docker

import (
	"encoding/json"
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
)

const (
	defaultDockerRuntime = "runc"
)

// Config defines a docker config file.
// TODO: This should not be public, but we need to access it from the tests in tools/container/docker
type Config map[string]interface{}

var _ engine.Interface = (*Config)(nil)

type dockerRuntime map[string]interface{}

var _ engine.RuntimeConfig = (*dockerRuntime)(nil)

// GetBinaryPath retrieves the path to the low-level runtime binary for a runtime.
// If no path is available, the empty string is returned.
func (d dockerRuntime) GetBinaryPath() string {
	if d == nil {
		return ""
	}

	path, _ := d["path"].(string)
	return path
}

// New creates a docker config with the specified options
func New(opts ...Option) (engine.Interface, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	if b.logger == nil {
		b.logger = logger.New()
	}

	return b.build()
}

// AddRuntime adds a new runtime to the docker config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	config := *c

	// Read the existing runtimes
	runtimes := make(map[string]interface{})
	if _, exists := config["runtimes"]; exists {
		runtimes = config["runtimes"].(map[string]interface{})
	}

	// Add / update the runtime definitions
	runtimes[name] = map[string]interface{}{
		"path": path,
		"args": []string{},
	}

	config["runtimes"] = runtimes

	if setAsDefault {
		config["default-runtime"] = name
	}

	*c = config
	return nil
}

// DefaultRuntime returns the default runtime for the docker config
func (c Config) DefaultRuntime() string {
	r, ok := c["default-runtime"].(string)
	if !ok {
		return ""
	}
	return r
}

// EnableCDI sets features.cdi to true in the docker config.
func (c *Config) EnableCDI() {
	if c == nil {
		return
	}
	config := *c

	features, ok := config["features"].(map[string]bool)
	if !ok {
		features = make(map[string]bool)
	}
	features["cdi"] = true

	config["features"] = features

	*c = config
}

// RemoveRuntime removes a runtime from the docker config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil {
		return nil
	}
	config := *c

	if _, exists := config["default-runtime"]; exists {
		defaultRuntime := config["default-runtime"].(string)
		if defaultRuntime == name {
			config["default-runtime"] = defaultDockerRuntime
		}
	}

	if _, exists := config["runtimes"]; exists {
		runtimes := config["runtimes"].(map[string]interface{})

		delete(runtimes, name)

		if len(runtimes) == 0 {
			delete(config, "runtimes")
		}
	}

	*c = config

	return nil
}

// Save writes the config to the specified path
func (c Config) Save(path string) (int64, error) {
	output, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return 0, fmt.Errorf("unable to convert to JSON: %v", err)
	}

	n, err := config.Raw(path).Write(output)
	return int64(n), err
}

// GetRuntimeConfig returns the runtime info of the runtime passed as input
func (c *Config) GetRuntimeConfig(name string) (engine.RuntimeConfig, error) {
	if c == nil {
		return nil, fmt.Errorf("config is nil")
	}

	cfg := *c

	var runtimes map[string]interface{}
	if _, ok := cfg["runtimes"]; ok {
		runtimes = cfg["runtimes"].(map[string]interface{})
		if r, ok := runtimes[name]; ok {
			dr := dockerRuntime(r.(map[string]interface{}))
			return &dr, nil
		}
	}
	return &dockerRuntime{}, nil
}

// String returns the string representation of the JSON config.
func (c Config) String() string {
	output, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Sprintf("invalid JSON: %v", err)
	}

	return string(output)
}
