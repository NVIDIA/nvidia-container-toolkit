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

	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
)

// LoadConfig loads the cri-o config from disk
func LoadConfig(config string) (*toml.Tree, error) {
	log.Infof("Loading config: %v", config)

	info, err := os.Stat(config)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	configFile := config
	if os.IsNotExist(err) {
		configFile = "/dev/null"
		log.Infof("Config file does not exist, creating new one")
	}

	cfg, err := toml.LoadFile(configFile)
	if err != nil {
		return nil, err
	}

	log.Infof("Successfully loaded config")

	return cfg, nil
}

// UpdateConfig updates the cri-o config to include the NVIDIA Container Runtime
func UpdateConfig(config *toml.Tree, runtimeClass string, runtimePath string, setAsDefault bool) error {
	switch runc := config.Get("crio.runtime.runtimes.runc").(type) {
	case *toml.Tree:
		runc, _ = toml.Load(runc.String())
		config.SetPath([]string{"crio", "runtime", "runtimes", runtimeClass}, runc)
	}

	config.SetPath([]string{"crio", "runtime", "runtimes", runtimeClass, "runtime_path"}, runtimePath)
	config.SetPath([]string{"crio", "runtime", "runtimes", runtimeClass, "runtime_type"}, "oci")

	if setAsDefault {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, runtimeClass)
	}

	return nil
}

// RevertConfig reverts the cri-o config to remove the NVIDIA Container Runtime
func RevertConfig(config *toml.Tree, runtimeClass string) error {
	if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		if runtimeClass == runtime {
			config.DeletePath([]string{"crio", "runtime", "default_runtime"})
		}
	}

	runtimeClassPath := []string{"crio", "runtime", "runtimes", runtimeClass}
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

	return nil
}

// FlushConfig flushes the updated/reverted config out to disk
func FlushConfig(config string, cfg *toml.Tree) error {
	log.Infof("Flushing config")

	output, err := cfg.ToTomlString()
	if err != nil {
		return fmt.Errorf("unable to convert to TOML: %v", err)
	}

	switch len(output) {
	case 0:
		err := os.Remove(config)
		if err != nil {
			return fmt.Errorf("unable to remove empty file: %v", err)
		}
		log.Infof("Config empty, removing file")
	default:
		f, err := os.Create(config)
		if err != nil {
			return fmt.Errorf("unable to open '%v' for writing: %v", config, err)
		}
		defer f.Close()

		_, err = f.WriteString(output)
		if err != nil {
			return fmt.Errorf("unable to write output: %v", err)
		}
	}

	log.Infof("Successfully flushed config")

	return nil
}
