/**
# Copyright (c) 2020-2021, NVIDIA CORPORATION.  All rights reserved.
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

package containerd

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

const (
	Name = "containerd"

	DefaultConfig      = "/etc/containerd/config.toml"
	DefaultSocket      = "/run/containerd/containerd.sock"
	DefaultRestartMode = "signal"

	defaultRuntmeType = "io.containerd.runc.v2"
)

// Options stores the containerd-specific options
type Options struct {
	useLegacyConfig bool
	runtimeType     string

	ContainerRuntimeModesCDIAnnotationPrefixes cli.StringSlice

	runtimeConfigOverrideJSON string
}

func Flags(opts *Options) []cli.Flag {
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name: "use-legacy-config",
			Usage: "Specify whether a legacy (pre v1.3) config should be used. " +
				"This ensures that a version 1 container config is created by default and that the " +
				"containerd.runtimes.default_runtime config section is used to define the default " +
				"runtime instead of container.default_runtime_name.",
			Destination: &opts.useLegacyConfig,
			EnvVars:     []string{"CONTAINERD_USE_LEGACY_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "runtime-type",
			Usage:       "The runtime_type to use for the configured runtime classes",
			Value:       defaultRuntmeType,
			Destination: &opts.runtimeType,
			EnvVars:     []string{"CONTAINERD_RUNTIME_TYPE"},
		},
		&cli.StringSliceFlag{
			Name:        "nvidia-container-runtime-modes.cdi.annotation-prefixes",
			Destination: &opts.ContainerRuntimeModesCDIAnnotationPrefixes,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODES_CDI_ANNOTATION_PREFIXES"},
		},
		&cli.StringFlag{
			Name:        "runtime-config-override",
			Destination: &opts.runtimeConfigOverrideJSON,
			Usage:       "specify additional runtime options as a JSON string. The paths are relative to the runtime config.",
			Value:       "{}",
			EnvVars:     []string{"RUNTIME_CONFIG_OVERRIDE", "CONTAINERD_RUNTIME_CONFIG_OVERRIDE"},
		},
	}

	return flags
}

// Setup updates a containerd configuration to include the nvidia-containerd-runtime and reloads it
func Setup(c *cli.Context, o *container.Options, co *Options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	cfg, err := getRuntimeConfig(o, co)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Configure(cfg)
	if err != nil {
		return fmt.Errorf("unable to configure containerd: %v", err)
	}

	err = RestartContainerd(o)
	if err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}

	log.Infof("Completed 'setup' for %v", c.App.Name)

	return nil
}

// Cleanup reverts a containerd configuration to remove the nvidia-containerd-runtime and reloads it
func Cleanup(c *cli.Context, o *container.Options, co *Options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	cfg, err := getRuntimeConfig(o, co)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Unconfigure(cfg)
	if err != nil {
		return fmt.Errorf("unable to unconfigure containerd: %v", err)
	}

	err = RestartContainerd(o)
	if err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}

	log.Infof("Completed 'cleanup' for %v", c.App.Name)

	return nil
}

// RestartContainerd restarts containerd depending on the value of restartModeFlag
func RestartContainerd(o *container.Options) error {
	return o.Restart("containerd", SignalContainerd)
}

// containerAnnotationsFromCDIPrefixes returns the container annotations to set for the given CDI prefixes.
func (o *Options) containerAnnotationsFromCDIPrefixes() []string {
	var annotations []string
	for _, prefix := range o.ContainerRuntimeModesCDIAnnotationPrefixes.Value() {
		annotations = append(annotations, prefix+"*")
	}

	return annotations
}

func (o *Options) runtimeConfigOverride() (map[string]interface{}, error) {
	if o.runtimeConfigOverrideJSON == "" {
		return nil, nil
	}

	runtimeOptions := make(map[string]interface{})
	if err := json.Unmarshal([]byte(o.runtimeConfigOverrideJSON), &runtimeOptions); err != nil {
		return nil, fmt.Errorf("failed to read %v as JSON: %w", o.runtimeConfigOverrideJSON, err)
	}

	return runtimeOptions, nil
}

func GetLowlevelRuntimePaths(o *container.Options, co *Options) ([]string, error) {
	cfg, err := getRuntimeConfig(o, co)
	if err != nil {
		return nil, fmt.Errorf("unable to load containerd config: %w", err)
	}
	return engine.GetBinaryPathsForRuntimes(cfg), nil
}

func getRuntimeConfig(o *container.Options, co *Options) (engine.Interface, error) {
	return containerd.New(
		containerd.WithPath(o.Config),
		containerd.WithConfigSource(
			toml.LoadFirst(
				containerd.CommandLineSource(o.HostRootMount),
				toml.FromFile(o.Config),
			),
		),
		containerd.WithRuntimeType(co.runtimeType),
		containerd.WithUseLegacyConfig(co.useLegacyConfig),
		containerd.WithContainerAnnotations(co.containerAnnotationsFromCDIPrefixes()...),
	)
}
