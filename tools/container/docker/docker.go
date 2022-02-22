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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	restartModeSignal = "signal"
	restartModeNone   = "none"

	nvidiaRuntimeName               = "nvidia"
	nvidiaRuntimeBinary             = "nvidia-container-runtime"
	nvidiaExperimentalRuntimeName   = "nvidia-experimental"
	nvidiaExperimentalRuntimeBinary = "nvidia-container-runtime-experimental"

	defaultConfig       = "/etc/docker/daemon.json"
	defaultSocket       = "/var/run/docker.sock"
	defaultSetAsDefault = true
	// defaultRuntimeName specifies the NVIDIA runtime to be use as the default runtime if setting the default runtime is enabled
	defaultRuntimeName = nvidiaRuntimeName
	defaultRestartMode = restartModeSignal

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
	config       string
	socket       string
	runtimeName  string
	setAsDefault bool
	runtimeDir   string
	restartMode  string
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
			Aliases:     []string{"c"},
			Usage:       "Path to docker config file",
			Value:       defaultConfig,
			Destination: &options.config,
			EnvVars:     []string{"DOCKER_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Aliases:     []string{"s"},
			Usage:       "Path to the docker socket file",
			Value:       defaultSocket,
			Destination: &options.socket,
			EnvVars:     []string{"DOCKER_SOCKET"},
		},
		// The flags below are only used by the 'setup' command.
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"r"},
			Usage:       "Specify the name of the `nvidia` runtime. If set-as-default is selected, the runtime is used as the default runtime.",
			Value:       defaultRuntimeName,
			Destination: &options.runtimeName,
			EnvVars:     []string{"DOCKER_RUNTIME_NAME"},
		},
		&cli.BoolFlag{
			Name:        "set-as-default",
			Aliases:     []string{"d"},
			Usage:       "Set the `nvidia` runtime as the default runtime. If --runtime-name is specified as `nvidia-experimental` the experimental runtime is set as the default runtime instead",
			Value:       defaultSetAsDefault,
			Destination: &options.setAsDefault,
			EnvVars:     []string{"DOCKER_SET_AS_DEFAULT"},
			Hidden:      true,
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how docker should be restarted; If 'none' is selected it will not be restarted [signal | none]",
			Value:       defaultRestartMode,
			Destination: &options.restartMode,
			EnvVars:     []string{"DOCKER_RESTART_MODE"},
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

	cfg, err := LoadConfig(o.config)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = UpdateConfig(cfg, o)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	err = FlushConfig(cfg, o.config)
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

	cfg, err := LoadConfig(o.config)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = RevertConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to update config: %v", err)
	}

	err = FlushConfig(cfg, o.config)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
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

// LoadConfig loads the docker config from disk
func LoadConfig(config string) (map[string]interface{}, error) {
	log.Infof("Loading config: %v", config)

	info, err := os.Stat(config)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	cfg := make(map[string]interface{})

	if os.IsNotExist(err) {
		log.Infof("Config file does not exist, creating new one")
		return cfg, nil
	}

	readBytes, err := ioutil.ReadFile(config)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %v", err)
	}

	reader := bytes.NewReader(readBytes)
	if err := json.NewDecoder(reader).Decode(&cfg); err != nil {
		return nil, err
	}

	log.Infof("Successfully loaded config")
	return cfg, nil
}

// UpdateConfig updates the docker config to include the nvidia runtimes
func UpdateConfig(config map[string]interface{}, o *options) error {
	defaultRuntime := o.getDefaultRuntime()
	if defaultRuntime != "" {
		config["default-runtime"] = defaultRuntime
	}

	runtimes := make(map[string]interface{})
	if _, exists := config["runtimes"]; exists {
		runtimes = config["runtimes"].(map[string]interface{})
	}

	for name, rt := range o.runtimes() {
		runtimes[name] = rt
	}

	config["runtimes"] = runtimes
	return nil
}

//RevertConfig reverts the docker config to remove the nvidia runtime
func RevertConfig(config map[string]interface{}) error {
	if _, exists := config["default-runtime"]; exists {
		defaultRuntime := config["default-runtime"].(string)
		if _, exists := nvidiaRuntimeBinaries[defaultRuntime]; exists {
			config["default-runtime"] = defaultDockerRuntime
		}
	}

	if _, exists := config["runtimes"]; exists {
		runtimes := config["runtimes"].(map[string]interface{})

		for name := range nvidiaRuntimeBinaries {
			delete(runtimes, name)
		}

		if len(runtimes) == 0 {
			delete(config, "runtimes")
		}
	}
	return nil
}

// FlushConfig flushes the updated/reverted config out to disk
func FlushConfig(cfg map[string]interface{}, config string) error {
	log.Infof("Flushing config")

	output, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to convert to JSON: %v", err)
	}

	switch len(output) {
	case 0:
		err := os.Remove(config)
		if err != nil {
			return fmt.Errorf("unable to remove empty file: %v", err)
		}
		log.Infof("Config empty, removing file")
	default:
		f, err := os.Create(config)
		if err != nil {
			return fmt.Errorf("unable to open %v for writing: %v", config, err)
		}
		defer f.Close()

		_, err = f.WriteString(string(output))
		if err != nil {
			return fmt.Errorf("unable to write output: %v", err)
		}
	}

	log.Infof("Successfully flushed config")

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

// getDefaultRuntime returns the default runtime for the configured options.
// If the configuration is invalid or the default runtimes should not be set
// the empty string is returned.
func (o options) getDefaultRuntime() string {
	if o.setAsDefault == false {
		return ""
	}

	return o.runtimeName
}

// runtimes returns the docker runtime definitions for the supported nvidia runtimes
// for the given options. This includes the path with the options runtimeDir applied
func (o options) runtimes() map[string]interface{} {
	runtimes := make(map[string]interface{})
	for r, bin := range o.getRuntimeBinaries() {
		runtimes[r] = map[string]interface{}{
			"path": bin,
			"args": []string{},
		}
	}
	return runtimes
}

// getRuntimeBinaries returns a map of runtime names to binary paths. This includes the
// renaming of the `nvidia` runtime as per the --runtime-class command line flag.
func (o options) getRuntimeBinaries() map[string]string {
	runtimeBinaries := make(map[string]string)

	for rt, bin := range nvidiaRuntimeBinaries {
		runtime := rt
		if o.runtimeName != "" && o.runtimeName != nvidiaExperimentalRuntimeName && runtime == defaultRuntimeName {
			runtime = o.runtimeName
		}

		runtimeBinaries[runtime] = filepath.Join(o.runtimeDir, bin)
	}

	return runtimeBinaries
}
