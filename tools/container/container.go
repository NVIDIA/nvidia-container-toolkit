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
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container/operator"
)

const (
	restartModeNone    = "none"
	restartModeSignal  = "signal"
	restartModeSystemd = "systemd"
)

// Options defines the shared options for the CLIs to configure containers runtimes.
type Options struct {
	Config        string
	Socket        string
	RuntimeName   string
	RuntimeDir    string
	SetAsDefault  bool
	RestartMode   string
	HostRootMount string
}

// ParseArgs parses the command line arguments to the CLI
func ParseArgs(c *cli.Context, o *Options) error {
	if o.RuntimeDir != "" {
		logrus.Debug("Runtime directory already set; ignoring arguments")
		return nil
	}

	args := c.Args()

	logrus.Infof("Parsing arguments: %v", args.Slice())
	if c.NArg() != 1 {
		return fmt.Errorf("incorrect number of arguments")
	}

	o.RuntimeDir = args.Get(0)

	logrus.Infof("Successfully parsed arguments")

	return nil
}

// Configure applies the options to the specified config
func (o Options) Configure(cfg engine.Interface) error {
	err := o.UpdateConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}
	return o.flush(cfg)
}

// Unconfigure removes the options from the specified config
func (o Options) Unconfigure(cfg engine.Interface) error {
	err := o.RevertConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}
	return o.flush(cfg)
}

// flush flushes the specified config to disk
func (o Options) flush(cfg engine.Interface) error {
	logrus.Infof("Flushing config to %v", o.Config)
	n, err := cfg.Save(o.Config)
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

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error restarting %v using systemd: %v", service, err)
	}

	return nil
}
