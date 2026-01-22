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

package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/operator"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

const (
	restartModeNone    = "none"
	restartModeSignal  = "signal"
	restartModeSystemd = "systemd"
)

// Options defines the shared options for the CLIs to configure containers runtimes.
type Options struct {
	DropInConfig         string
	DropInConfigHostPath string
	// TopLevelConfigPath stores the path to the top-level config for the runtime.
	TopLevelConfigPath string
	Socket             string
	// ExecutablePath specifies the path to the container runtime executable.
	// This is used to extract the current config, for example.
	// If a HostRootMount is specified, this path is relative to the host root
	// mount.
	ExecutablePath string
	// EnabledCDI indicates whether CDI should be enabled.
	EnableCDI     bool
	RuntimeName   string
	RuntimeDir    string
	SetAsDefault  bool
	RestartMode   string
	HostRootMount string

	ConfigSources []string
}

// Configure applies the options to the specified config
func (o Options) Configure(cfg engine.Interface) error {
	err := o.UpdateConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}
	return o.Flush(cfg)
}

// Unconfigure removes the options from the specified config
func (o Options) Unconfigure(cfg engine.Interface) error {
	err := o.RevertConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	if err := o.Flush(cfg); err != nil {
		return err
	}

	if o.DropInConfig == "" {
		return nil
	}
	// When a drop-in config is used, we remove the drop-in file explicitly.
	// This is require for cases where we may have to include other contents
	// in the drop-in file and as such it may not be empty when we flush it.
	err = os.Remove(o.DropInConfig)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove drop-in config file: %w", err)
	}

	return nil
}

// Flush flushes the specified config to disk
func (o Options) Flush(cfg engine.Interface) error {
	filepath := o.DropInConfig
	if filepath == "" {
		filepath = o.TopLevelConfigPath
	}
	logrus.Infof("Flushing config to %v", filepath)
	n, err := cfg.Save(filepath)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}
	if n == 0 {
		logrus.Infof("Config file is empty, removed")
	}
	return nil
}

// UpdateConfig updates the specified config to include the nvidia runtimes
func (o Options) UpdateConfig(cfg engine.Interface) error {
	runtimes := operator.GetRuntimes(
		operator.WithNvidiaRuntimeName(o.RuntimeName),
		operator.WithSetAsDefault(o.SetAsDefault),
		operator.WithRoot(o.RuntimeDir),
	)
	for name, runtime := range runtimes {
		err := cfg.AddRuntime(name, runtime.Path, runtime.SetAsDefault)
		if err != nil {
			return fmt.Errorf("failed to update runtime %q: %v", name, err)
		}
	}

	if o.EnableCDI {
		cfg.EnableCDI()
	}

	return nil
}

// RevertConfig reverts the specified config to remove the nvidia runtimes
func (o Options) RevertConfig(cfg engine.Interface) error {
	runtimes := operator.GetRuntimes(
		operator.WithNvidiaRuntimeName(o.RuntimeName),
		operator.WithSetAsDefault(o.SetAsDefault),
		operator.WithRoot(o.RuntimeDir),
	)
	for name := range runtimes {
		err := cfg.RemoveRuntime(name)
		if err != nil {
			return fmt.Errorf("failed to remove runtime %q: %v", name, err)
		}
	}

	return nil
}

// Restart restarts the specified service
func (o Options) Restart(service string, withSignal func(string) error) error {
	switch o.RestartMode {
	case restartModeNone:
		logrus.Warningf("Skipping restart of %v due to --restart-mode=%v", service, o.RestartMode)
		return nil
	case restartModeSignal:
		return withSignal(o.Socket)
	case restartModeSystemd:
		return o.SystemdRestart(service)
	}

	return fmt.Errorf("invalid restart mode specified: %v", o.RestartMode)
}

// SystemdRestart restarts the specified service using systemd
func (o Options) SystemdRestart(service string) error {
	var args []string
	var msg string
	if o.HostRootMount != "" {
		msg = " on host"
		args = append(args, "chroot", o.HostRootMount)
	}
	args = append(args, "systemctl", "restart", service)

	logrus.Infof("Restarting %v%v using systemd: %v", service, msg, args)

	args = oci.Escape(args)
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error restarting %v using systemd: %v", service, err)
	}

	return nil
}

// GetConfigLoaders returns the loaders for the requested config sources.
// Supported config sources can be specified as:
//
// * 'file[=path/to/file]': The specified file or the top-level config path is used.
// * command: The runtime-specific function supplied as an argument is used.
func (o Options) GetConfigLoaders(commandSourceFunc func(string, string) toml.Loader) ([]toml.Loader, error) {
	if len(o.ConfigSources) == 0 {
		return []toml.Loader{toml.Empty}, nil
	}
	var loaders []toml.Loader
	for _, configSource := range o.ConfigSources {
		parts := strings.SplitN(configSource, "=", 2)
		source := strings.TrimSpace(parts[0])
		switch source {
		case "file":
			fileSourcePath := o.TopLevelConfigPath
			if len(parts) > 1 {
				fileSourcePath = strings.TrimSpace(parts[1])
			}
			loaders = append(loaders, toml.FromFile(fileSourcePath))
		case "command":
			if commandSourceFunc == nil {
				logrus.Warnf("Ignoring command config source")
			}
			if len(parts) > 1 {
				logrus.Warnf("Ignoring additional command argument %q", parts[1])
			}
			loaders = append(loaders, commandSourceFunc(o.HostRootMount, o.ExecutablePath))
		default:
			return nil, fmt.Errorf("unsupported config source %q", configSource)
		}
	}
	return loaders, nil
}
