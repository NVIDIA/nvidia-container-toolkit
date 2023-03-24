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

package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container/operator"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	restartModeSignal = "signal"
	restartModeNone   = "none"

	nvidiaRuntimeName               = "nvidia"
	nvidiaRuntimeBinary             = "nvidia-container-runtime"
	nvidiaExperimentalRuntimeName   = "nvidia-experimental"
	nvidiaExperimentalRuntimeBinary = "nvidia-container-runtime.experimental"

	defaultConfig       = "/etc/docker/daemon.json"
	defaultSocket       = "/var/run/docker.sock"
	defaultSetAsDefault = true
	// defaultRuntimeName specifies the NVIDIA runtime to be use as the default runtime if setting the default runtime is enabled
	defaultRuntimeName   = nvidiaRuntimeName
	defaultRestartMode   = restartModeSignal
	defaultHostRootMount = "/host"

	reloadBackoff     = 5 * time.Second
	maxReloadAttempts = 6

	defaultDockerRuntime  = "runc"
	socketMessageToGetPID = "GET /info HTTP/1.0\r\n\r\n"
)

// nvidiaRuntimeBinaries defines a map of runtime names to binary names
var nvidiaRuntimeBinaries = map[string]string{
	nvidiaRuntimeName:             nvidiaRuntimeBinary,
	nvidiaExperimentalRuntimeName: nvidiaExperimentalRuntimeBinary,
}

// options stores the configuration from the command line or environment variables
type options struct {
	config        string
	socket        string
	runtimeName   string
	setAsDefault  bool
	runtimeDir    string
	restartMode   string
	hostRootMount string
}

func main() {
	options := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "docker"
	c.Usage = "Update docker config with the nvidia runtime"
	c.Version = "0.1.0"

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Trigger docker config to be updated"
	setup.ArgsUsage = "<runtime_dirname>"
	setup.Action = func(c *cli.Context) error {
		return Setup(c, &options)
	}

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Trigger any updates made to docker config to be undone"
	cleanup.ArgsUsage = "<runtime_dirname>"
	cleanup.Action = func(c *cli.Context) error {
		return Cleanup(c, &options)
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
			Usage:       "Path to docker config file",
			Value:       defaultConfig,
			Destination: &options.config,
			EnvVars:     []string{"DOCKER_CONFIG", "RUNTIME_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the docker socket file",
			Value:       defaultSocket,
			Destination: &options.socket,
			EnvVars:     []string{"DOCKER_SOCKET", "RUNTIME_SOCKET"},
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how docker should be restarted; If 'none' is selected it will not be restarted [signal | none]",
			Value:       defaultRestartMode,
			Destination: &options.restartMode,
			EnvVars:     []string{"DOCKER_RESTART_MODE", "RUNTIME_RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting docker using systemd",
			Value:       defaultHostRootMount,
			Destination: &options.hostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
			// Restart using systemd is currently not supported.
			// We hide this option for the time being.
			Hidden: true,
		},
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"runtime-class", "nvidia-runtime-name"},
			Usage:       "Specify the name of the `nvidia` runtime. If set-as-default is selected, the runtime is used as the default runtime.",
			Value:       defaultRuntimeName,
			Destination: &options.runtimeName,
			EnvVars:     []string{"DOCKER_RUNTIME_NAME", "NVIDIA_RUNTIME_NAME"},
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Usage:       "Set the `nvidia` runtime as the default runtime. If --runtime-name is specified as `nvidia-experimental` the experimental runtime is set as the default runtime instead",
			Value:       defaultSetAsDefault,
			Destination: &options.setAsDefault,
			EnvVars:     []string{"DOCKER_SET_AS_DEFAULT", "NVIDIA_RUNTIME_SET_AS_DEFAULT"},
			Hidden:      true,
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Errorf("Error running docker configuration: %v", err)
		os.Exit(1)
	}
}

// Setup updates docker configuration to include the nvidia runtime and reloads it
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	runtimeDir, err := ParseArgs(c)
	if err != nil {
		return fmt.Errorf("unable to parse args: %v", err)
	}
	o.runtimeDir = runtimeDir

	cfg, err := docker.New(
		docker.WithPath(o.config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = UpdateConfig(cfg, o)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	log.Infof("Flushing docker config to %v", o.config)
	_, err = cfg.Save(o.config)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to restart docker: %v", err)
	}

	log.Infof("Completed 'setup' for %v", c.App.Name)

	return nil
}

// Cleanup reverts docker configuration to remove the nvidia runtime and reloads it
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	_, err := ParseArgs(c)
	if err != nil {
		return fmt.Errorf("unable to parse args: %v", err)
	}

	cfg, err := docker.New(
		docker.WithPath(o.config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = RevertConfig(cfg, o)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	log.Infof("Flushing docker config to %v", o.config)
	n, err := cfg.Save(o.config)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}
	if n == 0 {
		log.Infof("Config file is empty, removed")
	}

	err = RestartDocker(o)
	if err != nil {
		return fmt.Errorf("unable to signal docker: %v", err)
	}

	log.Infof("Completed 'cleanup' for %v", c.App.Name)

	return nil
}

// ParseArgs parses the command line arguments to the CLI
func ParseArgs(c *cli.Context) (string, error) {
	args := c.Args()

	log.Infof("Parsing arguments: %v", args.Slice())
	if args.Len() != 1 {
		return "", fmt.Errorf("incorrect number of arguments")
	}
	runtimeDir := args.Get(0)
	log.Infof("Successfully parsed arguments")

	return runtimeDir, nil
}

// UpdateConfig updates the docker config to include the nvidia runtimes
func UpdateConfig(cfg engine.Interface, o *options) error {
	runtimes := operator.GetRuntimes(
		operator.WithNvidiaRuntimeName(o.runtimeName),
		operator.WithSetAsDefault(o.setAsDefault),
		operator.WithRoot(o.runtimeDir),
	)
	for name, runtime := range runtimes {
		err := cfg.AddRuntime(name, runtime.Path, runtime.SetAsDefault)
		if err != nil {
			return fmt.Errorf("failed to update runtime %q: %v", name, err)
		}
	}

	return nil
}

// RevertConfig reverts the docker config to remove the nvidia runtime
func RevertConfig(cfg engine.Interface, o *options) error {
	runtimes := operator.GetRuntimes(
		operator.WithNvidiaRuntimeName(o.runtimeName),
		operator.WithSetAsDefault(o.setAsDefault),
		operator.WithRoot(o.runtimeDir),
	)
	for name := range runtimes {
		err := cfg.RemoveRuntime(name)
		if err != nil {
			return fmt.Errorf("failed to remove runtime %q: %v", name, err)
		}
	}

	return nil
}

// RestartDocker restarts docker depending on the value of restartModeFlag
func RestartDocker(o *options) error {
	switch o.restartMode {
	case restartModeNone:
		log.Warnf("Skipping sending signal to docker due to --restart-mode=%v", o.restartMode)
	case restartModeSignal:
		err := SignalDocker(o.socket)
		if err != nil {
			return fmt.Errorf("unable to signal docker: %v", err)
		}
	default:
		return fmt.Errorf("invalid restart mode specified: %v", o.restartMode)
	}

	return nil
}

// SignalDocker sends a SIGHUP signal to docker daemon
func SignalDocker(socket string) error {
	log.Infof("Sending SIGHUP signal to docker")

	// Wrap the logic to perform the SIGHUP in a function so we can retry it on failure
	retriable := func() error {
		conn, err := net.Dial("unix", socket)
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
			return fmt.Errorf("unable to send SIGHUP to 'docker' process: %v", err)
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
		log.Warnf("Error signaling docker, attempt %v/%v: %v", i+1, maxReloadAttempts, err)
		time.Sleep(reloadBackoff)
	}
	if err != nil {
		log.Warnf("Max retries reached %v/%v, aborting", maxReloadAttempts, maxReloadAttempts)
		return err
	}

	log.Infof("Successfully signaled docker")

	return nil
}
