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

package toolkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvdevices"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	transformroot "github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform/root"
)

const (
	// DefaultNvidiaDriverRoot specifies the default NVIDIA driver run directory
	DefaultNvidiaDriverRoot = "/run/nvidia/driver"

	configFilename = "config.toml"
)

type cdiOptions struct {
	Enabled   bool
	outputDir string
	kind      string
	vendor    string
	class     string
}

type Options struct {
	DriverRoot        string
	DevRoot           string
	DriverRootCtrPath string
	DevRootCtrPath    string

	ContainerRuntimeMode     string
	ContainerRuntimeDebug    string
	ContainerRuntimeLogLevel string

	ContainerRuntimeModesCdiDefaultKind        string
	ContainerRuntimeModesCDIAnnotationPrefixes cli.StringSlice

	ContainerRuntimeRuntimes cli.StringSlice

	ContainerRuntimeHookSkipModeDetection bool

	ContainerCLIDebug string

	// CDI stores the CDI options for the toolkit.
	CDI cdiOptions

	createDeviceNodes cli.StringSlice

	acceptNVIDIAVisibleDevicesWhenUnprivileged bool
	acceptNVIDIAVisibleDevicesAsVolumeMounts   bool

	ignoreErrors bool

	optInFeatures cli.StringSlice
}

func Flags(opts *Options) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "driver-root",
			Aliases:     []string{"nvidia-driver-root"},
			Value:       DefaultNvidiaDriverRoot,
			Destination: &opts.DriverRoot,
			EnvVars:     []string{"NVIDIA_DRIVER_ROOT", "DRIVER_ROOT"},
		},
		&cli.StringFlag{
			Name:        "driver-root-ctr-path",
			Value:       DefaultNvidiaDriverRoot,
			Destination: &opts.DriverRootCtrPath,
			EnvVars:     []string{"DRIVER_ROOT_CTR_PATH"},
		},
		&cli.StringFlag{
			Name:        "dev-root",
			Usage:       "Specify the root where `/dev` is located. If this is not specified, the driver-root is assumed.",
			Destination: &opts.DevRoot,
			EnvVars:     []string{"NVIDIA_DEV_ROOT", "DEV_ROOT"},
		},
		&cli.StringFlag{
			Name:        "dev-root-ctr-path",
			Usage:       "Specify the root where `/dev` is located in the container. If this is not specified, the driver-root-ctr-path is assumed.",
			Destination: &opts.DevRootCtrPath,
			EnvVars:     []string{"DEV_ROOT_CTR_PATH"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime.debug",
			Aliases:     []string{"nvidia-container-runtime-debug"},
			Usage:       "Specify the location of the debug log file for the NVIDIA Container Runtime",
			Destination: &opts.ContainerRuntimeDebug,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_DEBUG"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime.log-level",
			Aliases:     []string{"nvidia-container-runtime-debug-log-level"},
			Destination: &opts.ContainerRuntimeLogLevel,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime.mode",
			Aliases:     []string{"nvidia-container-runtime-mode"},
			Destination: &opts.ContainerRuntimeMode,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODE"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime.modes.cdi.default-kind",
			Destination: &opts.ContainerRuntimeModesCdiDefaultKind,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODES_CDI_DEFAULT_KIND"},
		},
		&cli.StringSliceFlag{
			Name:        "nvidia-container-runtime.modes.cdi.annotation-prefixes",
			Destination: &opts.ContainerRuntimeModesCDIAnnotationPrefixes,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_MODES_CDI_ANNOTATION_PREFIXES"},
		},
		&cli.StringSliceFlag{
			Name:        "nvidia-container-runtime.runtimes",
			Destination: &opts.ContainerRuntimeRuntimes,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_RUNTIMES"},
		},
		&cli.BoolFlag{
			Name:        "nvidia-container-runtime-hook.skip-mode-detection",
			Value:       true,
			Destination: &opts.ContainerRuntimeHookSkipModeDetection,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_HOOK_SKIP_MODE_DETECTION"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-cli.debug",
			Aliases:     []string{"nvidia-container-cli-debug"},
			Usage:       "Specify the location of the debug log file for the NVIDIA Container CLI",
			Destination: &opts.ContainerCLIDebug,
			EnvVars:     []string{"NVIDIA_CONTAINER_CLI_DEBUG"},
		},
		&cli.BoolFlag{
			Name:        "accept-nvidia-visible-devices-envvar-when-unprivileged",
			Usage:       "Set the accept-nvidia-visible-devices-envvar-when-unprivileged config option",
			Value:       true,
			Destination: &opts.acceptNVIDIAVisibleDevicesWhenUnprivileged,
			EnvVars:     []string{"ACCEPT_NVIDIA_VISIBLE_DEVICES_ENVVAR_WHEN_UNPRIVILEGED"},
		},
		&cli.BoolFlag{
			Name:        "accept-nvidia-visible-devices-as-volume-mounts",
			Usage:       "Set the accept-nvidia-visible-devices-as-volume-mounts config option",
			Destination: &opts.acceptNVIDIAVisibleDevicesAsVolumeMounts,
			EnvVars:     []string{"ACCEPT_NVIDIA_VISIBLE_DEVICES_AS_VOLUME_MOUNTS"},
		},
		&cli.BoolFlag{
			Name:        "cdi-enabled",
			Aliases:     []string{"enable-cdi"},
			Usage:       "enable the generation of a CDI specification",
			Destination: &opts.CDI.Enabled,
			EnvVars:     []string{"CDI_ENABLED", "ENABLE_CDI"},
		},
		&cli.StringFlag{
			Name:        "cdi-output-dir",
			Usage:       "the directory where the CDI output files are to be written. If this is set to '', no CDI specification is generated.",
			Value:       "/var/run/cdi",
			Destination: &opts.CDI.outputDir,
			EnvVars:     []string{"CDI_OUTPUT_DIR"},
		},
		&cli.StringFlag{
			Name:        "cdi-kind",
			Usage:       "the vendor string to use for the generated CDI specification",
			Value:       "management.nvidia.com/gpu",
			Destination: &opts.CDI.kind,
			EnvVars:     []string{"CDI_KIND"},
		},
		&cli.BoolFlag{
			Name:        "ignore-errors",
			Usage:       "ignore errors when installing the NVIDIA Container toolkit. This is used for testing purposes only.",
			Hidden:      true,
			Destination: &opts.ignoreErrors,
		},
		&cli.StringSliceFlag{
			Name:        "create-device-nodes",
			Usage:       "(Only applicable with --cdi-enabled) specifies which device nodes should be created. If any one of the options is set to '' or 'none', no device nodes will be created.",
			Value:       cli.NewStringSlice("control"),
			Destination: &opts.createDeviceNodes,
			EnvVars:     []string{"CREATE_DEVICE_NODES"},
		},
		&cli.StringSliceFlag{
			Name:        "opt-in-features",
			Hidden:      true,
			Destination: &opts.optInFeatures,
			EnvVars:     []string{"NVIDIA_CONTAINER_TOOLKIT_OPT_IN_FEATURES"},
		},
	}

	return flags
}

// An Installer is used to install the NVIDIA Container Toolkit from the toolkit container.
type Installer struct {
	logger logger.Interface

	sourceRoot string
	// toolkitRoot specifies the destination path at which the toolkit is installed.
	toolkitRoot string
}

// NewInstaller creates an installer for the NVIDIA Container Toolkit.
func NewInstaller(opts ...Option) *Installer {
	i := &Installer{}
	for _, opt := range opts {
		opt(i)
	}

	if i.logger == nil {
		i.logger = logger.New()
	}

	return i
}

// ValidateOptions checks whether the specified options are valid
func (t *Installer) ValidateOptions(opts *Options) error {
	if t == nil {
		return fmt.Errorf("toolkit installer is not initilized")
	}
	if t.toolkitRoot == "" {
		return fmt.Errorf("invalid --toolkit-root option: %v", t.toolkitRoot)
	}

	vendor, class := parser.ParseQualifier(opts.CDI.kind)
	if err := parser.ValidateVendorName(vendor); err != nil {
		return fmt.Errorf("invalid CDI vendor name: %v", err)
	}
	if err := parser.ValidateClassName(class); err != nil {
		return fmt.Errorf("invalid CDI class name: %v", err)
	}
	opts.CDI.vendor = vendor
	opts.CDI.class = class

	if opts.CDI.Enabled && opts.CDI.outputDir == "" {
		t.logger.Warning("Skipping CDI spec generation (no output directory specified)")
		opts.CDI.Enabled = false
	}

	isDisabled := false
	for _, mode := range opts.createDeviceNodes.Value() {
		if mode != "" && mode != "none" && mode != "control" {
			return fmt.Errorf("invalid --create-device-nodes value: %v", mode)
		}
		if mode == "" || mode == "none" {
			isDisabled = true
			break
		}
	}
	if !opts.CDI.Enabled && !isDisabled {
		t.logger.Info("disabling device node creation since --cdi-enabled=false")
		isDisabled = true
	}
	if isDisabled {
		opts.createDeviceNodes = *cli.NewStringSlice()
	}

	return nil
}

// Install installs the components of the NVIDIA container toolkit.
// Any existing installation is removed.
func (t *Installer) Install(cli *cli.Context, opts *Options) error {
	if t == nil {
		return fmt.Errorf("toolkit installer is not initilized")
	}
	t.logger.Infof("Installing NVIDIA container toolkit to '%v'", t.toolkitRoot)

	t.logger.Infof("Removing existing NVIDIA container toolkit installation")
	err := os.RemoveAll(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error removing toolkit directory: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error removing toolkit directory: %v", err))
	}

	// Create a toolkit installer to actually install the toolkit components.
	toolkit, err := installer.New(
		installer.WithLogger(t.logger),
		installer.WithSourceRoot(t.sourceRoot),
		installer.WithIgnoreErrors(opts.ignoreErrors),
	)
	if err != nil {
		if !opts.ignoreErrors {
			return fmt.Errorf("could not create toolkit installer: %w", err)
		}
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("could not create toolkit installer: %w", err))
	}
	if err := toolkit.Install(t.toolkitRoot); err != nil {
		if !opts.ignoreErrors {
			return fmt.Errorf("could not install toolkit components: %w", err)
		}
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("could not install toolkit components: %w", err))
	}

	err = t.installToolkitConfig(cli, opts)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container toolkit config: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container toolkit config: %v", err))
	}

	err = t.createDeviceNodes(opts)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error creating device nodes: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error creating device nodes: %v", err))
	}

	nvidiaCDIHookPath := filepath.Join(t.toolkitRoot, "nvidia-cdi-hook")
	err = t.generateCDISpec(opts, nvidiaCDIHookPath)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error generating CDI specification: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error generating CDI specification: %v", err))
	}

	return nil
}

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

func (t *Installer) createDirectories(dir ...string) error {
	for _, d := range dir {
		t.logger.Infof("Creating directory '%v'", d)
		err := os.MkdirAll(d, 0755)
		if err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}
	return nil
}

func (t *Installer) createDeviceNodes(opts *Options) error {
	modes := opts.createDeviceNodes.Value()
	if len(modes) == 0 {
		return nil
	}

	devices, err := nvdevices.New(
		nvdevices.WithDevRoot(opts.DevRootCtrPath),
	)
	if err != nil {
		return fmt.Errorf("failed to create library: %v", err)
	}

	for _, mode := range modes {
		t.logger.Infof("Creating %v device nodes at %v", mode, opts.DevRootCtrPath)
		if mode != "control" {
			t.logger.Warningf("Unrecognised device mode: %v", mode)
			continue
		}
		if err := devices.CreateNVIDIAControlDevices(); err != nil {
			return fmt.Errorf("failed to create control device nodes: %v", err)
		}
	}
	return nil
}

// generateCDISpec generates a CDI spec for use in management containers
func (t *Installer) generateCDISpec(opts *Options, nvidiaCDIHookPath string) error {
	if !opts.CDI.Enabled {
		return nil
	}
	t.logger.Info("Generating CDI spec for management containers")
	cdilib, err := nvcdi.New(
		nvcdi.WithLogger(t.logger),
		nvcdi.WithMode(nvcdi.ModeManagement),
		nvcdi.WithDriverRoot(opts.DriverRootCtrPath),
		nvcdi.WithDevRoot(opts.DevRootCtrPath),
		nvcdi.WithNVIDIACDIHookPath(nvidiaCDIHookPath),
		nvcdi.WithVendor(opts.CDI.vendor),
		nvcdi.WithClass(opts.CDI.class),
	)
	if err != nil {
		return fmt.Errorf("failed to create CDI library for management containers: %v", err)
	}

	spec, err := cdilib.GetSpec()
	if err != nil {
		return fmt.Errorf("failed to genereate CDI spec for management containers: %v", err)
	}

	transformer := transformroot.NewDriverTransformer(
		transformroot.WithDriverRoot(opts.DriverRootCtrPath),
		transformroot.WithTargetDriverRoot(opts.DriverRoot),
		transformroot.WithDevRoot(opts.DevRootCtrPath),
		transformroot.WithTargetDevRoot(opts.DevRoot),
	)
	if err := transformer.Transform(spec.Raw()); err != nil {
		return fmt.Errorf("failed to transform driver root in CDI spec: %v", err)
	}

	name, err := cdi.GenerateNameForSpec(spec.Raw())
	if err != nil {
		return fmt.Errorf("failed to generate CDI name for management containers: %v", err)
	}
	err = spec.Save(filepath.Join(opts.CDI.outputDir, name))
	if err != nil {
		return fmt.Errorf("failed to save CDI spec for management containers: %v", err)
	}

	return nil
}
