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

package crio

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/ocihook"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

const (
	Name = "crio"

	defaultConfigMode = configModeConfig

	configModeConfig = "config"
	configModeHook   = "hook"

	// Hook-based settings
	defaultHooksDir     = "/usr/share/containers/oci/hooks.d"
	defaultHookFilename = "oci-nvidia-hook.json"

	// Config-based settings
	DefaultConfig       = "/etc/crio/crio.conf"
	DefaultDropInConfig = "/etc/crio/conf.d/99-nvidia.toml"

	DefaultSocket      = "/var/run/crio/crio.sock"
	DefaultRestartMode = "systemd"

	DefaultUnconfigureMode = "default-runtime-only"
)

// Options defines the cri-o specific options.
type Options struct {
	configMode string

	// hook-specific options
	hooksDir     string
	hookFilename string
}

func Flags(opts *Options) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "hooks-dir",
			Usage:       "path to the cri-o hooks directory",
			Value:       defaultHooksDir,
			Destination: &opts.hooksDir,
			Sources:     cli.EnvVars("CRIO_HOOKS_DIR"),
			DefaultText: defaultHooksDir,
		},
		&cli.StringFlag{
			Name:        "hook-filename",
			Usage:       "filename of the cri-o hook that will be created / removed in the hooks directory",
			Value:       defaultHookFilename,
			Destination: &opts.hookFilename,
			Sources:     cli.EnvVars("CRIO_HOOK_FILENAME"),
			DefaultText: defaultHookFilename,
		},
		&cli.StringFlag{
			Name:        "config-mode",
			Usage:       "the configuration mode to use. One of [hook | config]",
			Value:       defaultConfigMode,
			Destination: &opts.configMode,
			Sources:     cli.EnvVars("CRIO_CONFIG_MODE"),
		},
	}

	return flags
}

func (opts *Options) Validate(logger logger.Interface, _ *cli.Command) error {
	if opts.configMode != configModeConfig && opts.configMode != configModeHook {
		return fmt.Errorf("invalid cri-o config mode: %q", opts.configMode)
	}
	if opts.configMode == configModeHook {
		logger.Warningf("The %q config for cri-o is deprecated", opts.configMode)
	}
	return nil
}

// Setup installs the prestart hook required to launch GPU-enabled containers
func Setup(c *cli.Command, o *container.Options, co *Options) error {
	log.Infof("Starting 'setup' for %v", c.Name)

	switch co.configMode {
	case "hook":
		return setupHook(o, co)
	case "config":
		return setupConfig(o)
	default:
		return fmt.Errorf("invalid config-mode '%v'", co.configMode)
	}
}

// setupHook installs the prestart hook required to launch GPU-enabled containers
func setupHook(o *container.Options, co *Options) error {
	log.Infof("Installing prestart hook")

	hookPath := filepath.Join(co.hooksDir, co.hookFilename)
	err := ocihook.CreateHook(hookPath, filepath.Join(o.RuntimeDir, config.NVIDIAContainerRuntimeHookExecutable))
	if err != nil {
		return fmt.Errorf("error creating hook: %v", err)
	}

	return nil
}

// setupConfig updates the cri-o config for the NVIDIA container runtime
func setupConfig(o *container.Options) error {
	log.Infof("Updating config file")

	cfg, err := getRuntimeConfig(o)
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
func Cleanup(c *cli.Command, o *container.Options, co *Options) error {
	log.Infof("Starting 'cleanup' for %v", c.Name)

	switch co.configMode {
	case "hook":
		return cleanupHook(co)
	case "config":
		return cleanupConfig(o)
	default:
		return fmt.Errorf("invalid config-mode '%v'", co.configMode)
	}
}

// cleanupHook removes the prestart hook
func cleanupHook(co *Options) error {
	log.Infof("Removing prestart hook")

	hookPath := filepath.Join(co.hooksDir, co.hookFilename)
	err := os.Remove(hookPath)
	if err != nil {
		return fmt.Errorf("error removing hook '%v': %v", hookPath, err)
	}

	return nil
}

// cleanupConfig removes the NVIDIA container runtime from the cri-o config
func cleanupConfig(o *container.Options) error {
	log.Infof("Reverting config file modifications")

	if o.UnconfigureMode == "none" {
		log.Infof("Skipping cleanup as unconfigure mode is set to 'none'")
		return nil
	}

	cfg, err := getRuntimeConfig(o)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	if o.UnconfigureMode == "default-runtime-only" {
		log.Infof("Removing nvidia as the default runtime if configured")
		return o.UnsetDefaultRuntime(cfg)
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
func RestartCrio(o *container.Options) error {
	return o.Restart("crio", func(string) error { return fmt.Errorf("supporting crio via signal is unsupported") })
}

func GetLowlevelRuntimePaths(o *container.Options) ([]string, error) {
	cfg, err := getRuntimeConfig(o)
	if err != nil {
		return nil, fmt.Errorf("unable to load crio config: %w", err)
	}
	return engine.GetBinaryPathsForRuntimes(cfg), nil
}

func getRuntimeConfig(o *container.Options) (engine.Interface, error) {
	loaders, err := o.GetConfigLoaders(crio.CommandLineSource)
	if err != nil {
		return nil, err
	}
	return crio.New(
		crio.WithTopLevelConfigPath(o.TopLevelConfigPath),
		crio.WithConfigSource(
			toml.LoadFirst(
				loaders...,
			),
		),
	)
}
