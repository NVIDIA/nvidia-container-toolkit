/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package config

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/pelletier/go-toml"
)

const (
	configOverride = "XDG_CONFIG_HOME"
	configFilePath = "nvidia-container-runtime/config.toml"
)

var (
	// DefaultExecutableDir specifies the default path to use for executables if they cannot be located in the path.
	DefaultExecutableDir = "/usr/bin"

	// NVIDIAContainerRuntimeHookExecutable is the executable name for the NVIDIA Container Runtime Hook
	NVIDIAContainerRuntimeHookExecutable = "nvidia-container-runtime-hook"
	// NVIDIAContainerToolkitExecutable is the executable name for the NVIDIA Container Toolkit (an alias for the NVIDIA Container Runtime Hook)
	NVIDIAContainerToolkitExecutable = "nvidia-container-toolkit"

	configDir = "/etc/"
)

// Config represents the contents of the config.toml file for the NVIDIA Container Toolkit
// Note: This is currently duplicated by the HookConfig in cmd/nvidia-container-toolkit/hook_config.go
type Config struct {
	AcceptEnvvarUnprivileged bool `toml:"accept-nvidia-visible-devices-envvar-when-unprivileged"`

	NVIDIAContainerCLIConfig         ContainerCLIConfig `toml:"nvidia-container-cli"`
	NVIDIACTKConfig                  CTKConfig          `toml:"nvidia-ctk"`
	NVIDIAContainerRuntimeConfig     RuntimeConfig      `toml:"nvidia-container-runtime"`
	NVIDIAContainerRuntimeHookConfig RuntimeHookConfig  `toml:"nvidia-container-runtime-hook"`
}

// GetConfig sets up the config struct. Values are read from a toml file
// or set via the environment.
func GetConfig() (*Config, error) {
	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		configDir = XDGConfigDir
	}

	configFilePath := path.Join(configDir, configFilePath)

	tomlFile, err := os.Open(configFilePath)
	if err != nil {
		return getDefaultConfig(), nil
	}
	defer tomlFile.Close()

	cfg, err := loadConfigFrom(tomlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config values: %v", err)
	}

	return cfg, nil
}

// loadRuntimeConfigFrom reads the config from the specified Reader
func loadConfigFrom(reader io.Reader) (*Config, error) {
	toml, err := toml.LoadReader(reader)
	if err != nil {
		return nil, err
	}

	return getConfigFrom(toml)
}

// getConfigFrom reads the nvidia container runtime config from the specified toml Tree.
func getConfigFrom(toml *toml.Tree) (*Config, error) {
	cfg := getDefaultConfig()

	if toml == nil {
		return cfg, nil
	}

	cfg.AcceptEnvvarUnprivileged = toml.GetDefault("accept-nvidia-visible-devices-envvar-when-unprivileged", cfg.AcceptEnvvarUnprivileged).(bool)

	cfg.NVIDIAContainerCLIConfig = *getContainerCLIConfigFrom(toml)
	cfg.NVIDIACTKConfig = *getCTKConfigFrom(toml)
	runtimeConfig, err := getRuntimeConfigFrom(toml)
	if err != nil {
		return nil, fmt.Errorf("failed to load nvidia-container-runtime config: %v", err)
	}
	cfg.NVIDIAContainerRuntimeConfig = *runtimeConfig

	runtimeHookConfig, err := getRuntimeHookConfigFrom(toml)
	if err != nil {
		return nil, fmt.Errorf("failed to load nvidia-container-runtime-hook config: %v", err)
	}
	cfg.NVIDIAContainerRuntimeHookConfig = *runtimeHookConfig

	return cfg, nil
}

// getDefaultConfig defines the default values for the config
func getDefaultConfig() *Config {
	c := Config{
		AcceptEnvvarUnprivileged:     true,
		NVIDIAContainerCLIConfig:     *getDefaultContainerCLIConfig(),
		NVIDIACTKConfig:              *getDefaultCTKConfig(),
		NVIDIAContainerRuntimeConfig: *GetDefaultRuntimeConfig(),
	}

	return &c
}

// GetDefaultConfigToml returns the default config as a toml Tree.
func GetDefaultConfigToml() (*toml.Tree, error) {
	tree, err := toml.TreeFromMap(nil)
	if err != nil {
		return nil, err
	}

	tree.Set("disable-require", false)
	tree.SetWithComment("swarm-resource", "", true, "DOCKER_RESOURCE_GPU")
	tree.SetWithComment("accept-nvidia-visible-devices-envvar-when-unprivileged", "", true, true)
	tree.SetWithComment("accept-nvidia-visible-devices-as-volume-mounts", "", true, false)

	// nvidia-container-cli
	tree.SetWithComment("nvidia-container-cli.root", "", true, "/run/nvidia/driver")
	tree.SetWithComment("nvidia-container-cli.path", "", true, "/usr/bin/nvidia-container-cli")
	tree.Set("nvidia-container-cli.environment", []string{})
	tree.SetWithComment("nvidia-container-cli.debug", "", true, "/var/log/nvidia-container-toolkit.log")
	tree.SetWithComment("nvidia-container-cli.ldcache", "", true, "/etc/ld.so.cache")
	tree.Set("nvidia-container-cli.load-kmods", true)
	tree.SetWithComment("nvidia-container-cli.no-cgroups", "", true, false)
	tree.SetWithComment("nvidia-container-cli.user", "", true, "root:video")
	tree.Set("nvidia-container-cli.ldconfig", getLdConfigPath())

	// nvidia-container-runtime
	tree.SetWithComment("nvidia-container-runtime.debug", "", true, "/var/log/nvidia-container-runtime.log")
	tree.Set("nvidia-container-runtime.log-level", "info")

	commentLines := []string{
		"Specify the runtimes to consider. This list is processed in order and the PATH",
		"searched for matching executables unless the entry is an absolute path.",
	}
	tree.SetWithComment("nvidia-container-runtime.runtimes", strings.Join(commentLines, "\n "), false, []string{"docker-runc", "runc"})

	tree.Set("nvidia-container-runtime.mode", "auto")

	tree.Set("nvidia-container-runtime.modes.csv.mount-spec-path", "/etc/nvidia-container-runtime/host-files-for-container.d")
	tree.Set("nvidia-container-runtime.modes.cdi.default-kind", "nvidia.com/gpu")
	tree.Set("nvidia-container-runtime.modes.cdi.annotation-prefixes", []string{cdi.AnnotationPrefix})

	// nvidia-ctk
	tree.Set("nvidia-ctk.path", "nvidia-ctk")

	return tree, nil
}

func getLdConfigPath() string {
	return "@/sbin/ldconfig.real"
}
