/**
# Copyright 2024 NVIDIA CORPORATION
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

package runtime

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime/docker"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/toolkit"
)

const (
	defaultSetAsDefault = true
	// defaultRuntimeName specifies the NVIDIA runtime to be use as the default runtime if setting the default runtime is enabled
	defaultRuntimeName   = "nvidia"
	defaultHostRootMount = "/host"

	runtimeSpecificDefault = "RUNTIME_SPECIFIC_DEFAULT"
)

type Options struct {
	container.Options

	containerdOptions containerd.Options
	crioOptions       crio.Options
}

func Flags(opts *Options) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Path to the runtime config file",
			Value:       runtimeSpecificDefault,
			Destination: &opts.Config,
			EnvVars:     []string{"RUNTIME_CONFIG", "CONTAINERD_CONFIG", "DOCKER_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the runtime socket file",
			Value:       runtimeSpecificDefault,
			Destination: &opts.Socket,
			EnvVars:     []string{"RUNTIME_SOCKET", "CONTAINERD_SOCKET", "DOCKER_SOCKET"},
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how the runtime should be restarted; If 'none' is selected it will not be restarted [signal | systemd | none ]",
			Value:       runtimeSpecificDefault,
			Destination: &opts.RestartMode,
			EnvVars:     []string{"RUNTIME_RESTART_MODE"},
		},
		&cli.BoolFlag{
			Name:        "enable-cdi-in-runtime",
			Usage:       "Enable CDI in the configured runt	ime",
			Destination: &opts.EnableCDI,
			EnvVars:     []string{"RUNTIME_ENABLE_CDI"},
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting the runtime using systemd",
			Value:       defaultHostRootMount,
			Destination: &opts.HostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
		},
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"nvidia-runtime-name", "runtime-class"},
			Usage:       "Specify the name of the `nvidia` runtime. If set-as-default is selected, the runtime is used as the default runtime.",
			Value:       defaultRuntimeName,
			Destination: &opts.RuntimeName,
			EnvVars:     []string{"NVIDIA_RUNTIME_NAME", "CONTAINERD_RUNTIME_CLASS", "DOCKER_RUNTIME_NAME"},
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Usage:       "Set the `nvidia` runtime as the default runtime.",
			Value:       defaultSetAsDefault,
			Destination: &opts.SetAsDefault,
			EnvVars:     []string{"NVIDIA_RUNTIME_SET_AS_DEFAULT", "CONTAINERD_SET_AS_DEFAULT", "DOCKER_SET_AS_DEFAULT"},
			Hidden:      true,
		},
	}

	flags = append(flags, containerd.Flags(&opts.containerdOptions)...)
	flags = append(flags, crio.Flags(&opts.crioOptions)...)

	return flags
}

// ValidateOptions checks whether the specified options are valid
func ValidateOptions(c *cli.Context, opts *Options, runtime string, toolkitRoot string, to *toolkit.Options) error {
	// We set this option here to ensure that it is available in future calls.
	opts.RuntimeDir = toolkitRoot

	if !c.IsSet("enable-cdi-in-runtime") {
		opts.EnableCDI = to.CDI.Enabled
	}

	// Apply the runtime-specific config changes.
	switch runtime {
	case containerd.Name:
		if opts.Config == runtimeSpecificDefault {
			opts.Config = containerd.DefaultConfig
		}
		if opts.Socket == runtimeSpecificDefault {
			opts.Socket = containerd.DefaultSocket
		}
		if opts.RestartMode == runtimeSpecificDefault {
			opts.RestartMode = containerd.DefaultRestartMode
		}
	case crio.Name:
		if opts.Config == runtimeSpecificDefault {
			opts.Config = crio.DefaultConfig
		}
		if opts.Socket == runtimeSpecificDefault {
			opts.Socket = crio.DefaultSocket
		}
		if opts.RestartMode == runtimeSpecificDefault {
			opts.RestartMode = crio.DefaultRestartMode
		}
	case docker.Name:
		if opts.Config == runtimeSpecificDefault {
			opts.Config = docker.DefaultConfig
		}
		if opts.Socket == runtimeSpecificDefault {
			opts.Socket = docker.DefaultSocket
		}
		if opts.RestartMode == runtimeSpecificDefault {
			opts.RestartMode = docker.DefaultRestartMode
		}
	default:
		return fmt.Errorf("undefined runtime %v", runtime)
	}

	return nil
}

func Setup(c *cli.Context, opts *Options, runtime string) error {
	switch runtime {
	case containerd.Name:
		return containerd.Setup(c, &opts.Options, &opts.containerdOptions)
	case crio.Name:
		return crio.Setup(c, &opts.Options, &opts.crioOptions)
	case docker.Name:
		return docker.Setup(c, &opts.Options)
	default:
		return fmt.Errorf("undefined runtime %v", runtime)
	}
}

func Cleanup(c *cli.Context, opts *Options, runtime string) error {
	switch runtime {
	case containerd.Name:
		return containerd.Cleanup(c, &opts.Options, &opts.containerdOptions)
	case crio.Name:
		return crio.Cleanup(c, &opts.Options, &opts.crioOptions)
	case docker.Name:
		return docker.Cleanup(c, &opts.Options)
	default:
		return fmt.Errorf("undefined runtime %v", runtime)
	}
}

func GetLowlevelRuntimePaths(opts *Options, runtime string) ([]string, error) {
	switch runtime {
	case containerd.Name:
		return containerd.GetLowlevelRuntimePaths(&opts.Options, &opts.containerdOptions)
	case crio.Name:
		return crio.GetLowlevelRuntimePaths(&opts.Options)
	case docker.Name:
		return docker.GetLowlevelRuntimePaths(&opts.Options)
	default:
		return nil, fmt.Errorf("undefined runtime %v", runtime)
	}
}
