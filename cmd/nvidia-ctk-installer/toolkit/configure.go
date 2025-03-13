/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package toolkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

const (
	configFilename = "config.toml"
)

// installToolkitConfig installs the config file for the NVIDIA container toolkit ensuring
// that the settings are updated to match the desired install and nvidia driver directories.
func (t *Installer) installToolkitConfig(c *cli.Context, opts *Options) error {
	toolkitConfigDir := filepath.Join(t.toolkitRoot, ".config", "nvidia-container-runtime")
	toolkitConfigPath := filepath.Join(toolkitConfigDir, configFilename)

	t.logger.Infof("Installing NVIDIA container toolkit config '%v'", toolkitConfigPath)

	err := t.createDirectories(toolkitConfigDir)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("could not create required directories: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("could not create required directories: %v", err))
	}
	nvidiaContainerCliExecutablePath := filepath.Join(t.toolkitRoot, "nvidia-container-cli")
	nvidiaCTKPath := filepath.Join(t.toolkitRoot, "nvidia-ctk")
	nvidiaContainerRuntimeHookPath := filepath.Join(t.toolkitRoot, "nvidia-container-runtime-hook")

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("could not open source config file: %v", err)
	}

	targetConfig, err := os.Create(toolkitConfigPath)
	if err != nil {
		return fmt.Errorf("could not create target config file: %v", err)
	}
	defer targetConfig.Close()

	// Read the ldconfig path from the config as this may differ per platform
	// On ubuntu-based systems this ends in `.real`
	ldconfigPath := fmt.Sprintf("%s", cfg.GetDefault("nvidia-container-cli.ldconfig", "/sbin/ldconfig"))
	// Use the driver run root as the root:
	driverLdconfigPath := config.NormalizeLDConfigPath("@" + filepath.Join(opts.DriverRoot, strings.TrimPrefix(ldconfigPath, "@/")))

	configValues := map[string]interface{}{
		// Set the options in the root toml table
		"accept-nvidia-visible-devices-envvar-when-unprivileged": opts.acceptNVIDIAVisibleDevicesWhenUnprivileged,
		"accept-nvidia-visible-devices-as-volume-mounts":         opts.acceptNVIDIAVisibleDevicesAsVolumeMounts,
		// Set the nvidia-container-cli options
		"nvidia-container-cli.root":     opts.DriverRoot,
		"nvidia-container-cli.path":     nvidiaContainerCliExecutablePath,
		"nvidia-container-cli.ldconfig": driverLdconfigPath,
		// Set nvidia-ctk options
		"nvidia-ctk.path": nvidiaCTKPath,
		// Set the nvidia-container-runtime-hook options
		"nvidia-container-runtime-hook.path":                nvidiaContainerRuntimeHookPath,
		"nvidia-container-runtime-hook.skip-mode-detection": opts.ContainerRuntimeHookSkipModeDetection,
	}

	toolkitRuntimeList := opts.ContainerRuntimeRuntimes.Value()
	if len(toolkitRuntimeList) > 0 {
		configValues["nvidia-container-runtime.runtimes"] = toolkitRuntimeList
	}

	for _, optInFeature := range opts.optInFeatures.Value() {
		configValues["features."+optInFeature] = true
	}

	for key, value := range configValues {
		cfg.Set(key, value)
	}

	// Set the optional config options
	optionalConfigValues := map[string]interface{}{
		"nvidia-container-runtime.debug":                         opts.ContainerRuntimeDebug,
		"nvidia-container-runtime.log-level":                     opts.ContainerRuntimeLogLevel,
		"nvidia-container-runtime.mode":                          opts.ContainerRuntimeMode,
		"nvidia-container-runtime.modes.cdi.annotation-prefixes": opts.ContainerRuntimeModesCDIAnnotationPrefixes,
		"nvidia-container-runtime.modes.cdi.default-kind":        opts.ContainerRuntimeModesCdiDefaultKind,
		"nvidia-container-runtime.runtimes":                      opts.ContainerRuntimeRuntimes,
		"nvidia-container-cli.debug":                             opts.ContainerCLIDebug,
	}

	for key, value := range optionalConfigValues {
		if !c.IsSet(key) {
			t.logger.Infof("Skipping unset option: %v", key)
			continue
		}
		if value == nil {
			t.logger.Infof("Skipping option with nil value: %v", key)
			continue
		}

		switch v := value.(type) {
		case string:
			if v == "" {
				continue
			}
		case cli.StringSlice:
			if len(v.Value()) == 0 {
				continue
			}
			value = v.Value()
		default:
			t.logger.Warningf("Unexpected type for option %v=%v: %T", key, value, v)
		}

		cfg.Set(key, value)
	}

	if _, err := cfg.WriteTo(targetConfig); err != nil {
		return fmt.Errorf("error writing config: %v", err)
	}

	os.Stdout.WriteString("Using config:\n")
	if _, err = cfg.WriteTo(os.Stdout); err != nil {
		t.logger.Warningf("Failed to output config to STDOUT: %v", err)
	}

	return nil
}
