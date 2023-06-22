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
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/pelletier/go-toml"
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

	return Load(configFilePath)
}

// Load loads the config from the specified file path.
func Load(configFilePath string) (*Config, error) {
	if configFilePath == "" {
		return getDefault()
	}

	tomlFile, err := os.Open(configFilePath)
	if err != nil {
		return getDefault()
	}
	defer tomlFile.Close()

	cfg, err := LoadFrom(tomlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config values: %v", err)
	}

	return cfg, nil
}

// LoadFrom reads the config from the specified Reader
func LoadFrom(reader io.Reader) (*Config, error) {
	var tree *toml.Tree
	if reader != nil {
		toml, err := toml.LoadReader(reader)
		if err != nil {
			return nil, err
		}
		tree = toml
	}

	return getFromTree(tree)
}

// getFromTree reads the nvidia container runtime config from the specified toml Tree.
func getFromTree(toml *toml.Tree) (*Config, error) {
	cfg, err := getDefault()
	if err != nil {
		return nil, err
	}
	if toml == nil {
		return cfg, nil
	}

	if err := toml.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return cfg, nil
}

// getDefault defines the default values for the config
func getDefault() (*Config, error) {
	d := Config{
		AcceptEnvvarUnprivileged: true,
		NVIDIAContainerCLIConfig: ContainerCLIConfig{
			LoadKmods: true,
			Ldconfig:  getLdConfigPath(),
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
	if _, err := os.Stat("/sbin/ldconfig.real"); err == nil {
		return "@/sbin/ldconfig.real"
	}
	return "@/sbin/ldconfig"
}

// getCommentedUserGroup returns whether the nvidia-container-cli user and group config option should be commented.
func getCommentedUserGroup() bool {
	uncommentIf := map[string]bool{
		"suse":     true,
		"opensuse": true,
	}

	idsLike := getDistIDLike()
	for _, id := range idsLike {
		if uncommentIf[id] {
			return false
		}
	}
	return true
}

// getDistIDLike returns the ID_LIKE field from /etc/os-release.
func getDistIDLike() []string {
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

func (c Config) asCommentedToml() (*toml.Tree, error) {
	contents, err := toml.Marshal(c)
	if err != nil {
		return nil, err
	}
	asToml, err := toml.LoadBytes(contents)
	if err != nil {
		return nil, err
	}

	commentedDefaults := map[string]interface{}{
		"swarm-resource": "DOCKER_RESOURCE_GPU",
		"accept-nvidia-visible-devices-envvar-when-unprivileged": true,
		"accept-nvidia-visible-devices-as-volume-mounts":         false,
		"nvidia-container-cli.root":                              "/run/nvidia/driver",
		"nvidia-container-cli.path":                              "/usr/bin/nvidia-container-cli",
		"nvidia-container-cli.debug":                             "/var/log/nvidia-container-toolkit.log",
		"nvidia-container-cli.ldcache":                           "/etc/ld.so.cache",
		"nvidia-container-cli.no-cgroups":                        false,
		"nvidia-container-cli.user":                              "root:video",
		"nvidia-container-runtime.debug":                         "/var/log/nvidia-container-runtime.log",
	}
	for k, v := range commentedDefaults {
		set := asToml.Get(k)
		fmt.Printf("k=%v v=%+v set=%+v\n", k, v, set)
		if !shouldComment(k, v, set) {
			continue
		}
		fmt.Printf("set=%+v v=%+v\n", set, v)
		asToml.SetWithComment(k, "", true, v)
	}

	return asToml, nil
}

func shouldComment(key string, value interface{}, set interface{}) bool {
	if key == "nvidia-container-cli.user" && !getCommentedUserGroup() {
		return false
	}
	if key == "nvidia-container-runtime.debug" && set == "/dev/null" {
		return true
	}
	if set == nil || value == set {
		return true
	}

	return false
}

// Save writes the config to the specified writer.
func (c Config) Save(w io.Writer) (int64, error) {
	asToml, err := c.asCommentedToml()
	if err != nil {
		return 0, err
	}

	enc := toml.NewEncoder(w).Indentation("")
	if err := enc.Encode(asToml); err != nil {
		return 0, fmt.Errorf("invalid config: %v", err)
	}
	return 0, nil
}
