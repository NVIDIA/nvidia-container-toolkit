package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
)

const (
	driverPath = "/run/nvidia/driver"
)

// hookConfig wraps the toolkit config.
// This allows for functions to be defined on the local type.
type hookConfig struct {
	sync.Mutex
	*config.Config
	containerConfig *containerConfig
}

// loadConfig loads the required paths for the hook config.
func loadConfig() (*config.Config, error) {
	configFilePath, required := getConfigFilePath()
	cfg, err := config.New(
		config.WithConfigFile(configFilePath),
		config.WithRequired(true),
	)
	if err == nil {
		return cfg.Config()
	} else if os.IsNotExist(err) && !required {
		return config.GetDefault()
	}
	return nil, fmt.Errorf("couldn't open required configuration file: %v", err)
}

func getConfigFilePath() (string, bool) {
	if configFromFlag := *configflag; configFromFlag != "" {
		return configFromFlag, true
	}
	if configFromEnvvar := os.Getenv(config.FilePathOverrideEnvVar); configFromEnvvar != "" {
		return configFromEnvvar, true
	}
	return config.GetConfigFilePath(), false
}

func getHookConfig() (*hookConfig, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	config := &hookConfig{Config: cfg}

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

	return config, nil
}

// getConfigOption returns the toml config option associated with the
// specified struct field.
func (c *hookConfig) getConfigOption(fieldName string) string {
	t := reflect.TypeOf(&c)
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
func (c *hookConfig) getSwarmResourceEnvvars() []string {
	if c == nil || c.SwarmResource == "" {
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

// nvidiaContainerCliCUDACompatModeFlags returns required --cuda-compat-mode
// flag(s) depending on the hook and runtime configurations.
func (c *hookConfig) nvidiaContainerCliCUDACompatModeFlags() []string {
	var flag string
	switch c.NVIDIAContainerRuntimeConfig.Modes.Legacy.CUDACompatMode {
	case config.CUDACompatModeLdconfig:
		flag = "--cuda-compat-mode=ldconfig"
	case config.CUDACompatModeMount:
		flag = "--cuda-compat-mode=mount"
	case config.CUDACompatModeDisabled, config.CUDACompatModeHook:
		flag = "--cuda-compat-mode=disabled"
	default:
		if !c.Features.AllowCUDACompatLibsFromContainer.IsEnabled() {
			flag = "--cuda-compat-mode=disabled"
		}
	}

	if flag == "" {
		return nil
	}
	return []string{flag}
}

func (c *hookConfig) assertModeIsLegacy() error {
	if c.NVIDIAContainerRuntimeHookConfig.SkipModeDetection {
		return nil
	}

	mr := info.NewRuntimeModeResolver(
		info.WithLogger(&logInterceptor{}),
		info.WithImage(&c.containerConfig.Image),
		info.WithDefaultMode(info.LegacyRuntimeMode),
	)

	mode := mr.ResolveRuntimeMode(c.NVIDIAContainerRuntimeConfig.Mode)
	if mode == "legacy" {
		return nil
	}
	return fmt.Errorf("invoking the NVIDIA Container Runtime Hook directly (e.g. specifying the docker --gpus flag) is not supported. Please use the NVIDIA Container Runtime (e.g. specify the --runtime=nvidia flag) instead")
}
