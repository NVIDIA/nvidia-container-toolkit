/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/ocihook"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container"
)

const (
	defaultConfigMode = "hook"

	// Hook-based settings
	defaultHooksDir     = "/usr/share/containers/oci/hooks.d"
	defaultHookFilename = "oci-nvidia-hook.json"

	// Config-based settings
	defaultConfig        = "/etc/crio/crio.conf"
	defaultSocket        = "/var/run/crio/crio.sock"
	defaultRuntimeClass  = "nvidia"
	defaultSetAsDefault  = true
	defaultRestartMode   = "systemd"
	defaultHostRootMount = "/host"
)

// options stores the configuration from the command linek or environment variables
type options struct {
	container.Options

	configMode string

	// hook-specific options
	hooksDir     string
	hookFilename string
}

func main() {
	options := options{}

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "crio"
	c.Usage = "Update cri-o hooks to include the NVIDIA runtime hook"
	c.Version = info.GetVersionString()

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Configure cri-o for NVIDIA GPU containers"
	setup.ArgsUsage = "<toolkit_dirname>"
	setup.Action = func(c *cli.Context) error {
		return Setup(c, &options)
	}
	setup.Before = func(c *cli.Context) error {
		return container.ParseArgs(c, &options.Options)
	}

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Remove the NVIDIA-specific cri-o configuration"
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
			Usage:       "Path to the cri-o config file",
			Value:       defaultConfig,
			Destination: &options.Config,
			EnvVars:     []string{"RUNTIME_CONFIG", "CRIO_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to the crio socket file",
			Value:       "",
			Destination: &options.Socket,
			EnvVars:     []string{"RUNTIME_SOCKET", "CRIO_SOCKET"},
			// Note: We hide this option since restarting cri-o via a socket is not supported.
			Hidden: true,
		},
		&cli.StringFlag{
			Name:        "restart-mode",
			Usage:       "Specify how cri-o should be restarted;  If 'none' is selected, it will not be restarted [systemd | none]",
			Value:       defaultRestartMode,
			Destination: &options.RestartMode,
			EnvVars:     []string{"RUNTIME_RESTART_MODE", "CRIO_RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:        "runtime-name",
			Aliases:     []string{"nvidia-runtime-name", "runtime-class"},
			Usage:       "The name of the runtime class to set for the nvidia-container-runtime",
			Value:       defaultRuntimeClass,
			Destination: &options.RuntimeName,
			EnvVars:     []string{"NVIDIA_RUNTIME_NAME", "CRIO_RUNTIME_CLASS"},
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
			EnvVars:     []string{"NVIDIA_RUNTIME_SET_AS_DEFAULT", "CRIO_SET_AS_DEFAULT"},
			Hidden:      true,
		},
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "Specify the path to the host root to be used when restarting crio using systemd",
			Value:       defaultHostRootMount,
			Destination: &options.HostRootMount,
			EnvVars:     []string{"HOST_ROOT_MOUNT"},
		},
		&cli.StringFlag{
			Name:        "hooks-dir",
			Usage:       "path to the cri-o hooks directory",
			Value:       defaultHooksDir,
			Destination: &options.hooksDir,
			EnvVars:     []string{"CRIO_HOOKS_DIR"},
			DefaultText: defaultHooksDir,
		},
		&cli.StringFlag{
			Name:        "hook-filename",
			Usage:       "filename of the cri-o hook that will be created / removed in the hooks directory",
			Value:       defaultHookFilename,
			Destination: &options.hookFilename,
			EnvVars:     []string{"CRIO_HOOK_FILENAME"},
			DefaultText: defaultHookFilename,
		},
		&cli.StringFlag{
			Name:        "config-mode",
			Usage:       "the configuration mode to use. One of [hook | config]",
			Value:       defaultConfigMode,
			Destination: &options.configMode,
			EnvVars:     []string{"CRIO_CONFIG_MODE"},
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Fatal(fmt.Errorf("error: %v", err))
	}
}

// Setup installs the prestart hook required to launch GPU-enabled containers
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	switch o.configMode {
	case "hook":
		return setupHook(o)
	case "config":
		return setupConfig(o)
	default:
		return fmt.Errorf("invalid config-mode '%v'", o.configMode)
	}
}

// setupHook installs the prestart hook required to launch GPU-enabled containers
func setupHook(o *options) error {
	log.Infof("Installing prestart hook")

	hookPath := filepath.Join(o.hooksDir, o.hookFilename)
	err := ocihook.CreateHook(hookPath, filepath.Join(o.RuntimeDir, config.NVIDIAContainerRuntimeHookExecutable))
	if err != nil {
		return fmt.Errorf("error creating hook: %v", err)
	}

	return nil
}

// setupConfig updates the cri-o config for the NVIDIA container runtime
func setupConfig(o *options) error {
	log.Infof("Updating config file")

	cfg, err := crio.New(
		crio.WithPath(o.Config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Configure(cfg)
	if err != nil {
		return fmt.Errorf("unable to configure cri-o: %v", err)
	}

	err = RestartCrio(o)
	if err != nil {
		return fmt.Errorf("unable to restart crio: %v", err)
	}

	return nil
}

// Cleanup removes the specified prestart hook
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	switch o.configMode {
	case "hook":
		return cleanupHook(o)
	case "config":
		return cleanupConfig(o)
	default:
		return fmt.Errorf("invalid config-mode '%v'", o.configMode)
	}
}

// cleanupHook removes the prestart hook
func cleanupHook(o *options) error {
	log.Infof("Removing prestart hook")

	hookPath := filepath.Join(o.hooksDir, o.hookFilename)
	err := os.Remove(hookPath)
	if err != nil {
		return fmt.Errorf("error removing hook '%v': %v", hookPath, err)
	}

	return nil
}

// cleanupConfig removes the NVIDIA container runtime from the cri-o config
func cleanupConfig(o *options) error {
	log.Infof("Reverting config file modifications")

	cfg, err := crio.New(
		crio.WithPath(o.Config),
	)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	err = o.Unconfigure(cfg)
	if err != nil {
		return fmt.Errorf("unable to unconfigure cri-o: %v", err)
	}

	err = RestartCrio(o)
	if err != nil {
		return fmt.Errorf("unable to restart crio: %v", err)
	}

	return nil
}

// RestartCrio restarts crio depending on the value of restartModeFlag
func RestartCrio(o *options) error {
	return o.Restart("crio", func(string) error { return fmt.Errorf("supporting crio via signal is unsupported") })
}
