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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

// LoadConfig loads the docker config from disk
func LoadConfig(configFilePath string) (map[string]interface{}, error) {
	log.Infof("Loading docker config from %v", configFilePath)

	info, err := os.Stat(configFilePath)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	cfg := make(map[string]interface{})

	if os.IsNotExist(err) {
		log.Infof("Config file does not exist, creating new one")
		return cfg, nil
	}

	readBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %v", err)
	}

	reader := bytes.NewReader(readBytes)
	if err := json.NewDecoder(reader).Decode(&cfg); err != nil {
		return nil, err
	}

	log.Infof("Successfully loaded config")
	return cfg, nil
}

// UpdateConfig updates the docker config to include the nvidia runtimes
func UpdateConfig(config map[string]interface{}, runtimeName string, runtimePath string, setAsDefault bool) error {
	// Read the existing runtimes
	runtimes := make(map[string]interface{})
	if _, exists := config["runtimes"]; exists {
		runtimes = config["runtimes"].(map[string]interface{})
	}

	// Add / update the runtime definitions
	runtimes[runtimeName] = map[string]interface{}{
		"path": runtimePath,
		"args": []string{},
	}

	// Update the runtimes definition
	if len(runtimes) > 0 {
		config["runtimes"] = runtimes
	}

	if setAsDefault {
		config["default-runtime"] = runtimeName
	}

	return nil
}

// FlushConfig flushes the updated/reverted config out to disk
func FlushConfig(cfg map[string]interface{}, configFilePath string) error {
	log.Infof("Flushing docker config to %v", configFilePath)

	output, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to convert to JSON: %v", err)
	}

	switch len(output) {
	case 0:
		err := os.Remove(configFilePath)
		if err != nil {
			return fmt.Errorf("unable to remove empty file: %v", err)
		}
		log.Infof("Config empty, removing file")
	default:
		f, err := os.Create(configFilePath)
		if err != nil {
			return fmt.Errorf("unable to open %v for writing: %v", configFilePath, err)
		}
		defer f.Close()

		_, err = f.WriteString(string(output))
		if err != nil {
			return fmt.Errorf("unable to write output: %v", err)
		}
	}

	log.Infof("Successfully flushed config")

	return nil
}
