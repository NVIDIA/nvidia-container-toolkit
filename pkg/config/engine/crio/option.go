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

package crio

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"

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
		empty := toml.Tree{}
		return (*Config)(&empty), nil
	}
	if b.logger == nil {
		b.logger = logger.New()
	}

	return b.loadConfig(b.path)
}

// loadConfig loads the cri-o config from disk
func (b *builder) loadConfig(config string) (*Config, error) {
	b.logger.Infof("Loading config: %v", config)

	info, err := os.Stat(config)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	if os.IsNotExist(err) {
		b.logger.Infof("Config file does not exist; using empty config")
		config = "/dev/null"
	} else {
		b.logger.Infof("Loading config from %v", config)
	}

	cfg, err := toml.LoadFile(config)
	if err != nil {
		return nil, err
	}

	b.logger.Infof("Successfully loaded config")

	return (*Config)(cfg), nil
}
