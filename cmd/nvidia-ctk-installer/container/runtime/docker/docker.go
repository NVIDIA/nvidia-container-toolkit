/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
	"fmt"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
)

const (
	Name = "docker"

	DefaultConfig      = "/etc/docker/daemon.json"
	DefaultSocket      = "/var/run/docker.sock"
	DefaultRestartMode = "signal"
)

type Options struct{}

func Flags(opts *Options) []cli.Flag {
	return nil
}

// Setup updates docker configuration to include the nvidia runtime and reloads it
func Setup(c *cli.Context, o *container.Options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	cfg, err := getRuntimeConfig(o)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Configure(cfg)
	if err != nil {
		return fmt.Errorf("unable to configure docker: %v", err)
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to restart docker: %v", err)
	}

	log.Infof("Completed 'setup' for %v", c.App.Name)

	return nil
}

// Cleanup reverts docker configuration to remove the nvidia runtime and reloads it
func Cleanup(c *cli.Context, o *container.Options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	cfg, err := getRuntimeConfig(o)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Unconfigure(cfg)
	if err != nil {
		return fmt.Errorf("unable to unconfigure docker: %v", err)
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to signal docker: %v", err)
	}

	log.Infof("Completed 'cleanup' for %v", c.App.Name)

	return nil
}

// RestartDocker restarts docker depending on the value of restartModeFlag
func RestartDocker(o *container.Options) error {
	return o.Restart("docker", SignalDocker)
}

func GetLowlevelRuntimePaths(o *container.Options) ([]string, error) {
	cfg, err := docker.New(
		docker.WithPath(o.Config),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load docker config: %w", err)
	}
	return engine.GetBinaryPathsForRuntimes(cfg), nil
}

func getRuntimeConfig(o *container.Options) (engine.Interface, error) {
	return docker.New(
		docker.WithPath(o.Config),
	)
}
