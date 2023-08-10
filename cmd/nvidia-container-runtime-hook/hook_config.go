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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

const (
	configPath = "/etc/nvidia-container-runtime/config.toml"
	driverPath = "/run/nvidia/driver"
)

var defaultPaths = [...]string{
	path.Join(driverPath, configPath),
	configPath,
}

// HookConfig : options for the nvidia-container-runtime-hook.
type HookConfig config.Config

func getDefaultHookConfig() (HookConfig, error) {
	defaultCfg, err := config.GetDefault()
	if err != nil {
		return HookConfig{}, err
	}

	return *(*HookConfig)(defaultCfg), nil
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

	allSupportedDriverCapabilities := image.SupportedDriverCapabilities
	if config.SupportedDriverCapabilities == "all" {
		config.SupportedDriverCapabilities = allSupportedDriverCapabilities.String()
	}
	configuredCapabilities := image.NewDriverCapabilities(config.SupportedDriverCapabilities)
	// We ensure that the configured value is a subset of all supported capabilities
	if !allSupportedDriverCapabilities.IsSuperset(configuredCapabilities) {
		configName := config.getConfigOption("SupportedDriverCapabilities")
		log.Panicf("Invalid value for config option '%v'; %v (supported: %v)\n", configName, config.SupportedDriverCapabilities, allSupportedDriverCapabilities.String())
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
	if c.SwarmResource == "" {
		return nil
	}

	candidates := strings.Split(c.SwarmResource, ",")

	var envvars []string
	for _, c := range candidates {
		trimmed := strings.TrimSpace(c)
		if len(trimmed) > 0 {
			envvars = append(envvars, trimmed)
		}
	}

	return envvars
}
