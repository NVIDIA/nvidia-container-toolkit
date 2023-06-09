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

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	restartModeSignal  = "signal"
	restartModeSystemd = "systemd"
	restartModeNone    = "none"

	nvidiaRuntimeName               = "nvidia"
	nvidiaRuntimeBinary             = "nvidia-container-runtime"
	nvidiaExperimentalRuntimeName   = "nvidia-experimental"
	nvidiaExperimentalRuntimeBinary = "nvidia-container-runtime.experimental"

	defaultConfig        = "/etc/containerd/config.toml"
	defaultSocket        = "/run/containerd/containerd.sock"
	defaultRuntimeClass  = "nvidia"
	defaultRuntmeType    = "io.containerd.runc.v2"
	defaultSetAsDefault  = true
	defaultRestartMode   = restartModeSignal
	defaultHostRootMount = "/host"

	reloadBackoff     = 5 * time.Second
	maxReloadAttempts = 6

	socketMessageToGetPID = ""
)

// nvidiaRuntimeBinaries defines a map of runtime names to binary names
var nvidiaRuntimeBinaries = map[string]string{
	nvidiaRuntimeName:             nvidiaRuntimeBinary,
	nvidiaExperimentalRuntimeName: nvidiaExperimentalRuntimeBinary,
}

// options stores the configuration from the command line or environment variables
type options struct {
	container.Options

	// containerd-specific options
	useLegacyConfig bool
	runtimeType     string

	ContainerRuntimeModesCDIAnnotationPrefixes cli.StringSlice
}

func main() {
	options := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "containerd"
	c.Usage = "Update a containerd config with the nvidia-container-runtime"
	c.Version = "0.1.0"

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Trigger a containerd config to be updated"
	setup.ArgsUsage = "<runtime_dirname>"
	setup.Action = func(c *cli.Context) error {
		return Setup(c, &options)
	}
	setup.Before = func(c *cli.Context) error {
		return container.ParseArgs(c, &options.Options)
	}

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Trigger any updates made to a containerd config to be undone"
	cleanup.ArgsUsage = "<runtime_dirname>"
	cleanup.Action = func(c *cli.Context) error {
		return Cleanup(c, &options)
	}
	cleanup.Before = func(c *cli.Context) error {
		return container.ParseArgs(c, &options.Options)
	}

	// Register the subcommands with the top-level CLI
	c.Commands = []*cli.Command{
		&setup,
		&cleanup,
	}

	// Setup common flags across both subcommands. All subcommands get the same
	// set of flags even if they don't use some of them. This is so that we
	// only require the user to specify one set of flags for both 'startup'
	// and 'cleanup' to simplify things.
	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Path to the containerd config file",
			Value:       defaultConfig,
			Destination: &options.Config,
			EnvVars:     []string{"CONTAINERD_CONFIG", "RUNTIME_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the containerd socket file",
			Value:       defaultSocket,
			Destination: &options.Socket,
			EnvVars:     []string{"CONTAINERD_SOCKET", "RUNTIME_SOCKET"},
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how containerd should be restarted;  If 'none' is selected, it will not be restarted [signal | systemd | none]",
			Value:       defaultRestartMode,
			Destination: &options.RestartMode,
			EnvVars:     []string{"CONTAINERD_RESTART_MODE", "RUNTIME_RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:        "runtime-class",
			Aliases:     []string{"runtime-name", "nvidia-runtime-name"},
			Usage:       "The name of the runtime class to set for the nvidia-container-runtime",
			Value:       defaultRuntimeClass,
			Destination: &options.RuntimeName,
			EnvVars:     []string{"CONTAINERD_RUNTIME_CLASS", "NVIDIA_RUNTIME_NAME"},
		},
		&cli.StringFlag{
			Name:        "nvidia-runtime-dir",
			Aliases:     []string{"runtime-dir"},
			Usage:       "The path where the nvidia-container-runtime binaries are located. If this is not specified, the first argument will be used instead",
			Destination: &options.RuntimeDir,
			EnvVars:     []string{"NVIDIA_RUNTIME_DIR"},
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Usage:       "Set nvidia-container-runtime as the default runtime",
			Value:       defaultSetAsDefault,
			Destination: &options.SetAsDefault,
			EnvVars:     []string{"CONTAINERD_SET_AS_DEFAULT", "NVIDIA_RUNTIME_SET_AS_DEFAULT"},
			Hidden:      true,
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting containerd using systemd",
			Value:       defaultHostRootMount,
			Destination: &options.HostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
		},
		&cli.BoolFlag{
			Name:        "use-legacy-config",
			Usage:       "Specify whether a legacy (pre v1.3) config should be used",
			Destination: &options.useLegacyConfig,
			EnvVars:     []string{"CONTAINERD_USE_LEGACY_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "runtime-type",
			Usage:       "The runtime_type to use for the configured runtime classes",
			Value:       defaultRuntmeType,
			Destination: &options.runtimeType,
			EnvVars:     []string{"CONTAINERD_RUNTIME_TYPE"},
		},
		&cli.StringSliceFlag{
			Name:        "nvidia-container-runtime-modes.cdi.annotation-prefixes",
			Destination: &options.ContainerRuntimeModesCDIAnnotationPrefixes,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODES_CDI_ANNOTATION_PREFIXES"},
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Fatal(fmt.Errorf("Error: %v", err))
	}
}

// Setup updates a containerd configuration to include the nvidia-containerd-runtime and reloads it
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	cfg, err := containerd.New(
		containerd.WithPath(o.Config),
		containerd.WithRuntimeType(o.runtimeType),
		containerd.WithUseLegacyConfig(o.useLegacyConfig),
		containerd.WithContainerAnnotations(o.containerAnnotationsFromCDIPrefixes()...),
	)
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
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	cfg, err := containerd.New(
		containerd.WithPath(o.Config),
		containerd.WithRuntimeType(o.runtimeType),
		containerd.WithUseLegacyConfig(o.useLegacyConfig),
		containerd.WithContainerAnnotations(o.containerAnnotationsFromCDIPrefixes()...),
	)
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
func RestartContainerd(o *options) error {
	switch o.RestartMode {
	case restartModeNone:
		log.Warnf("Skipping sending signal to containerd due to --restart-mode=%v", o.RestartMode)
		return nil
	case restartModeSignal:
		err := SignalContainerd(o)
		if err != nil {
			return fmt.Errorf("unable to signal containerd: %v", err)
		}
	case restartModeSystemd:
		return RestartContainerdSystemd(o.HostRootMount)
	default:
		return fmt.Errorf("Invalid restart mode specified: %v", o.RestartMode)
	}

	return nil
}

// SignalContainerd sends a SIGHUP signal to the containerd daemon
func SignalContainerd(o *options) error {
	log.Infof("Sending SIGHUP signal to containerd")

	// Wrap the logic to perform the SIGHUP in a function so we can retry it on failure
	retriable := func() error {
		conn, err := net.Dial("unix", o.Socket)
		if err != nil {
			return fmt.Errorf("unable to dial: %v", err)
		}
		defer conn.Close()

		sconn, err := conn.(*net.UnixConn).SyscallConn()
		if err != nil {
			return fmt.Errorf("unable to get syscall connection: %v", err)
		}

		err1 := sconn.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_PASSCRED, 1)
		})
		if err1 != nil {
			return fmt.Errorf("unable to issue call on socket fd: %v", err1)
		}
		if err != nil {
			return fmt.Errorf("unable to SetsockoptInt on socket fd: %v", err)
		}

		_, _, err = conn.(*net.UnixConn).WriteMsgUnix([]byte(socketMessageToGetPID), nil, nil)
		if err != nil {
			return fmt.Errorf("unable to WriteMsgUnix on socket fd: %v", err)
		}

		oob := make([]byte, 1024)
		_, oobn, _, _, err := conn.(*net.UnixConn).ReadMsgUnix(nil, oob)
		if err != nil {
			return fmt.Errorf("unable to ReadMsgUnix on socket fd: %v", err)
		}

		oob = oob[:oobn]
		scm, err := syscall.ParseSocketControlMessage(oob)
		if err != nil {
			return fmt.Errorf("unable to ParseSocketControlMessage from message received on socket fd: %v", err)
		}

		ucred, err := syscall.ParseUnixCredentials(&scm[0])
		if err != nil {
			return fmt.Errorf("unable to ParseUnixCredentials from message received on socket fd: %v", err)
		}

		err = syscall.Kill(int(ucred.Pid), syscall.SIGHUP)
		if err != nil {
			return fmt.Errorf("unable to send SIGHUP to 'containerd' process: %v", err)
		}

		return nil
	}

	// Try to send a SIGHUP up to maxReloadAttempts times
	var err error
	for i := 0; i < maxReloadAttempts; i++ {
		err = retriable()
		if err == nil {
			break
		}
		if i == maxReloadAttempts-1 {
			break
		}
		log.Warnf("Error signaling containerd, attempt %v/%v: %v", i+1, maxReloadAttempts, err)
		time.Sleep(reloadBackoff)
	}
	if err != nil {
		log.Warnf("Max retries reached %v/%v, aborting", maxReloadAttempts, maxReloadAttempts)
		return err
	}

	log.Infof("Successfully signaled containerd")

	return nil
}

// RestartContainerdSystemd restarts containerd using systemctl
func RestartContainerdSystemd(hostRootMount string) error {
	log.Infof("Restarting containerd using systemd and host root mounted at %v", hostRootMount)

	command := "chroot"
	args := []string{hostRootMount, "systemctl", "restart", "containerd"}

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error restarting containerd using systemd: %v", err)
	}

	return nil
}

// containerAnnotationsFromCDIPrefixes returns the container annotations to set for the given CDI prefixes.
func (o *options) containerAnnotationsFromCDIPrefixes() []string {
	var annotations []string
	for _, prefix := range o.ContainerRuntimeModesCDIAnnotationPrefixes.Value() {
		annotations = append(annotations, prefix+"*")
	}

	return annotations
}
