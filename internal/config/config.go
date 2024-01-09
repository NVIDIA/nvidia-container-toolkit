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
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

const (
	configOverride = "XDG_CONFIG_HOME"
	configFilePath = "nvidia-container-runtime/config.toml"

	nvidiaCTKExecutable      = "nvidia-ctk"
	nvidiaCTKDefaultFilePath = "/usr/bin/nvidia-ctk"

	nvidiaContainerRuntimeHookExecutable  = "nvidia-container-runtime-hook"
	nvidiaContainerRuntimeHookDefaultPath = "/usr/bin/nvidia-container-runtime-hook"
)

var (
	// DefaultExecutableDir specifies the default path to use for executables if they cannot be located in the path.
	DefaultExecutableDir = "/usr/bin"

	// NVIDIAContainerRuntimeHookExecutable is the executable name for the NVIDIA Container Runtime Hook
	NVIDIAContainerRuntimeHookExecutable = "nvidia-container-runtime-hook"
	// NVIDIAContainerToolkitExecutable is the executable name for the NVIDIA Container Toolkit (an alias for the NVIDIA Container Runtime Hook)
	NVIDIAContainerToolkitExecutable = "nvidia-container-toolkit"
)

// Config represents the contents of the config.toml file for the NVIDIA Container Toolkit
// Note: This is currently duplicated by the HookConfig in cmd/nvidia-container-toolkit/hook_config.go
type Config struct {
	DisableRequire                 bool   `toml:"disable-require"`
	SwarmResource                  string `toml:"swarm-resource"`
	AcceptEnvvarUnprivileged       bool   `toml:"accept-nvidia-visible-devices-envvar-when-unprivileged"`
	AcceptDeviceListAsVolumeMounts bool   `toml:"accept-nvidia-visible-devices-as-volume-mounts"`
	SupportedDriverCapabilities    string `toml:"supported-driver-capabilities"`

	NVIDIAContainerCLIConfig         ContainerCLIConfig `toml:"nvidia-container-cli"`
	NVIDIACTKConfig                  CTKConfig          `toml:"nvidia-ctk"`
	NVIDIAContainerRuntimeConfig     RuntimeConfig      `toml:"nvidia-container-runtime"`
	NVIDIAContainerRuntimeHookConfig RuntimeHookConfig  `toml:"nvidia-container-runtime-hook"`
}

// GetConfigFilePath returns the path to the config file for the configured system
func GetConfigFilePath() string {
	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		return filepath.Join(XDGConfigDir, configFilePath)
	}

	return filepath.Join("/etc", configFilePath)
}

// GetConfig sets up the config struct. Values are read from a toml file
// or set via the environment.
func GetConfig() (*Config, error) {
	cfg, err := New(
		WithConfigFile(GetConfigFilePath()),
	)
	if err != nil {
		return nil, err
	}

	return cfg.Config()
}

// GetDefault defines the default values for the config
func GetDefault() (*Config, error) {
	d := Config{
		AcceptEnvvarUnprivileged:    true,
		SupportedDriverCapabilities: image.SupportedDriverCapabilities.String(),
		NVIDIAContainerCLIConfig: ContainerCLIConfig{
			LoadKmods: true,
			Ldconfig:  getLdConfigPath(),
			User:      getUserGroup(),
		},
		NVIDIACTKConfig: CTKConfig{
			Path: nvidiaCTKExecutable,
		},
		NVIDIAContainerRuntimeConfig: RuntimeConfig{
			DebugFilePath: "/dev/null",
			LogLevel:      "info",
			Runtimes:      []string{"docker-runc", "runc"},
			Mode:          "auto",
			Modes: modesConfig{
				CSV: csvModeConfig{
					MountSpecPath: "/etc/nvidia-container-runtime/host-files-for-container.d",
				},
				CDI: cdiModeConfig{
					DefaultKind:        "nvidia.com/gpu",
					AnnotationPrefixes: []string{cdi.AnnotationPrefix},
					SpecDirs:           cdi.DefaultSpecDirs,
				},
			},
		},
		NVIDIAContainerRuntimeHookConfig: RuntimeHookConfig{
			Path: NVIDIAContainerRuntimeHookExecutable,
		},
	}
	return &d, nil
}

func getLdConfigPath() string {
	return NormalizeLDConfigPath("@/sbin/ldconfig")
}

func getUserGroup() string {
	if isSuse() {
		return "root:video"
	}
	return ""
}

// isSuse returns whether a SUSE-based distribution was detected.
func isSuse() bool {
	suseDists := map[string]bool{
		"suse":     true,
		"opensuse": true,
	}

	idsLike := getDistIDLike()
	for _, id := range idsLike {
		if suseDists[id] {
			return true
		}
	}
	return false
}

// getDistIDLike returns the ID_LIKE field from /etc/os-release.
// We can override this for testing.
var getDistIDLike = func() []string {
	releaseFile, err := os.Open("/etc/os-release")
	if err != nil {
		return nil
	}
	defer releaseFile.Close()

	scanner := bufio.NewScanner(releaseFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID_LIKE=") {
			value := strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
			return strings.Split(value, " ")
		}
	}
	return nil
}

// ResolveNVIDIACTKPath resolves the path to the nvidia-ctk binary.
// This executable is used in hooks and needs to be an absolute path.
// If the path is specified as an absolute path, it is used directly
// without checking for existence of an executable at that path.
func ResolveNVIDIACTKPath(logger logger.Interface, nvidiaCTKPath string) string {
	return resolveWithDefault(
		logger,
		"NVIDIA Container Toolkit CLI",
		nvidiaCTKPath,
		nvidiaCTKDefaultFilePath,
	)
}

// ResolveNVIDIAContainerRuntimeHookPath resolves the path the nvidia-container-runtime-hook binary.
func ResolveNVIDIAContainerRuntimeHookPath(logger logger.Interface, nvidiaContainerRuntimeHookPath string) string {
	return resolveWithDefault(
		logger,
		"NVIDIA Container Runtime Hook",
		nvidiaContainerRuntimeHookPath,
		nvidiaContainerRuntimeHookDefaultPath,
	)
}

// resolveWithDefault resolves the path to the specified binary.
// If an absolute path is specified, it is used directly without searching for the binary.
// If the binary cannot be found in the path, the specified default is used instead.
func resolveWithDefault(logger logger.Interface, label string, path string, defaultPath string) string {
	if filepath.IsAbs(path) {
		logger.Debugf("Using specified %v path %v", label, path)
		return path
	}

	if path == "" {
		path = filepath.Base(defaultPath)
	}
	logger.Debugf("Locating %v as %v", label, path)
	lookup := lookup.NewExecutableLocator(logger, "")

	resolvedPath := defaultPath
	targets, err := lookup.Locate(path)
	if err != nil {
		logger.Warningf("Failed to locate %v: %v", path, err)
	} else {
		logger.Debugf("Found %v candidates: %v", path, targets)
		resolvedPath = targets[0]
	}
	logger.Debugf("Using %v path %v", label, path)

	return resolvedPath
}
