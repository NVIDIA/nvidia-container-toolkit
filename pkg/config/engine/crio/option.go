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
	"os/exec"

	"github.com/pelletier/go-toml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type builder struct {
	logger        logger.Interface
	path          string
	hostRootMount string
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

// WithHostRootMount sets the host root mount for the config builder
func WithHostRootMount(hostRootMount string) Option {
	return func(b *builder) {
		b.hostRootMount = hostRootMount
	}
}

func (b *builder) build() (*Config, error) {
	if b.logger == nil {
		b.logger = logger.New()
	}
	if b.path == "" {
		empty := toml.Tree{}
		c := Config{
			Tree:   &empty,
			Logger: b.logger,
		}
		return &c, nil
	}

	return b.loadConfig()
}

// loadConfig loads the cri-o config from disk
func (b *builder) loadConfig() (*Config, error) {

	var args []string
	if b.hostRootMount != "" {
		args = append(args, "chroot", b.hostRootMount)
	}
	args = append(args, "crio", "status", "config")

	b.logger.Infof("Getting current crio config")

	// TODO: Can we harden this so that there is less risk of command injection
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing crio status command: %w", err)
	}
	tomlConfig, err := toml.LoadBytes(output)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling crio config toml: %w", err)
	}

	b.logger.Infof("Successfully loaded crio config")

	c := Config{
		Tree:   tomlConfig,
		Logger: b.logger,
	}
	return &c, nil
}
