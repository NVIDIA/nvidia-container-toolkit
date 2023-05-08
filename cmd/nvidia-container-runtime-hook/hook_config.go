package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

const (
	configPath = "/etc/nvidia-container-runtime/config.toml"
	driverPath = "/run/nvidia/driver"
)

var defaultPaths = [...]string{
	path.Join(driverPath, configPath),
	configPath,
}

// CLIConfig : options for nvidia-container-cli.
type CLIConfig struct {
	Root        *string  `toml:"root"`
	Path        *string  `toml:"path"`
	Environment []string `toml:"environment"`
	Debug       *string  `toml:"debug"`
	Ldcache     *string  `toml:"ldcache"`
	LoadKmods   bool     `toml:"load-kmods"`
	NoPivot     bool     `toml:"no-pivot"`
	NoCgroups   bool     `toml:"no-cgroups"`
	User        *string  `toml:"user"`
	Ldconfig    *string  `toml:"ldconfig"`
}

// HookConfig : options for the nvidia-container-runtime-hook.
type HookConfig struct {
	DisableRequire                 bool               `toml:"disable-require"`
	SwarmResource                  *string            `toml:"swarm-resource"`
	AcceptEnvvarUnprivileged       bool               `toml:"accept-nvidia-visible-devices-envvar-when-unprivileged"`
	AcceptDeviceListAsVolumeMounts bool               `toml:"accept-nvidia-visible-devices-as-volume-mounts"`
	SupportedDriverCapabilities    DriverCapabilities `toml:"supported-driver-capabilities"`

	NvidiaContainerCLI         CLIConfig                `toml:"nvidia-container-cli"`
	NVIDIAContainerRuntime     config.RuntimeConfig     `toml:"nvidia-container-runtime"`
	NVIDIAContainerRuntimeHook config.RuntimeHookConfig `toml:"nvidia-container-runtime-hook"`
}

func getDefaultHookConfig() (HookConfig, error) {
	rtConfig, err := config.GetDefaultRuntimeConfig()
	if err != nil {
		return HookConfig{}, err
	}

	rtHookConfig, err := config.GetDefaultRuntimeHookConfig()
	if err != nil {
		return HookConfig{}, err
	}

	c := HookConfig{
		DisableRequire:                 false,
		SwarmResource:                  nil,
		AcceptEnvvarUnprivileged:       true,
		AcceptDeviceListAsVolumeMounts: false,
		SupportedDriverCapabilities:    allDriverCapabilities,
		NvidiaContainerCLI: CLIConfig{
			Root:        nil,
			Path:        nil,
			Environment: []string{},
			Debug:       nil,
			Ldcache:     nil,
			LoadKmods:   true,
			NoPivot:     false,
			NoCgroups:   false,
			User:        nil,
			Ldconfig:    nil,
		},
		NVIDIAContainerRuntime:     *rtConfig,
		NVIDIAContainerRuntimeHook: *rtHookConfig,
	}

	return c, nil
}

func getHookConfig() (*HookConfig, error) {
	var err error
	var config HookConfig

	if len(*configflag) > 0 {
		config, err = getDefaultHookConfig()
		if err != nil {
			return nil, fmt.Errorf("couldn't get default configuration: %v", err)
		}
		_, err = toml.DecodeFile(*configflag, &config)
		if err != nil {
			return nil, fmt.Errorf("couldn't open configuration file: %v", err)
		}
	} else {
		for _, p := range defaultPaths {
			config, err = getDefaultHookConfig()
			if err != nil {
				return nil, fmt.Errorf("couldn't get default configuration: %v", err)
			}
			_, err = toml.DecodeFile(p, &config)
			if err == nil {
				break
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("couldn't open default configuration file: %v", err)
			}
		}
	}

	if config.SupportedDriverCapabilities == all {
		config.SupportedDriverCapabilities = allDriverCapabilities
	}
	// We ensure that the supported-driver-capabilites option is a subset of allDriverCapabilities
	if intersection := allDriverCapabilities.Intersection(config.SupportedDriverCapabilities); intersection != config.SupportedDriverCapabilities {
		configName := config.getConfigOption("SupportedDriverCapabilities")
		log.Panicf("Invalid value for config option '%v'; %v (supported: %v)\n", configName, config.SupportedDriverCapabilities, allDriverCapabilities)
	}

	return &config, nil
}

// getConfigOption returns the toml config option associated with the
// specified struct field.
func (c HookConfig) getConfigOption(fieldName string) string {
	t := reflect.TypeOf(c)
	f, ok := t.FieldByName(fieldName)
	if !ok {
		return fieldName
	}
	v, ok := f.Tag.Lookup("toml")
	if !ok {
		return fieldName
	}
	return v
}

// getSwarmResourceEnvvars returns the swarm resource envvars for the config.
func (c *HookConfig) getSwarmResourceEnvvars() []string {
	if c.SwarmResource == nil {
		return nil
	}

	candidates := strings.Split(*c.SwarmResource, ",")

	var envvars []string
	for _, c := range candidates {
		trimmed := strings.TrimSpace(c)
		if len(trimmed) > 0 {
			envvars = append(envvars, trimmed)
		}
	}

	return envvars
}
