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

package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type builder struct {
	logger logger.Interface
	path   string
}

// Option defines a function that can be used to configure the config builder
type Option func(*builder)

// WithLogger sets the logger for the config builder
func WithLogger(logger logger.Interface) Option {
	return func(b *builder) {
		b.logger = logger
	}
}

// WithPath sets the path for the config builder
func WithPath(path string) Option {
	return func(b *builder) {
		b.path = path
	}
}

func (b *builder) build() (*Config, error) {
	if b.path == "" {
		empty := make(Config)
		return &empty, nil
	}

	return b.loadConfig(b.path)
}

// loadConfig loads the docker config from disk
func (b *builder) loadConfig(config string) (*Config, error) {
	info, err := os.Stat(config)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	cfg := make(Config)

	if os.IsNotExist(err) {
		b.logger.Infof("Config file does not exist; using empty config")
		return &cfg, nil
	}

	b.logger.Infof("Loading config from %v", config)
	readBytes, err := os.ReadFile(config)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %v", err)
	}

	reader := bytes.NewReader(readBytes)
	if err := json.NewDecoder(reader).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
