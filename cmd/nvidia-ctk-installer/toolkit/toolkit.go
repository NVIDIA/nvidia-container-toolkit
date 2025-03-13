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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/system/nvdevices"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	transformroot "github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform/root"
)

const (
	// DefaultNvidiaDriverRoot specifies the default NVIDIA driver run directory
	DefaultNvidiaDriverRoot = "/run/nvidia/driver"

	nvidiaContainerCliSource         = "/usr/bin/nvidia-container-cli"
	nvidiaContainerRuntimeHookSource = "/usr/bin/nvidia-container-runtime-hook"

	nvidiaContainerToolkitConfigSource = "/etc/nvidia-container-runtime/config.toml"
	configFilename                     = "config.toml"
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
	fileInstaller
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

	toolkitConfigDir := filepath.Join(t.toolkitRoot, ".config", "nvidia-container-runtime")
	toolkitConfigPath := filepath.Join(toolkitConfigDir, configFilename)

	err = t.createDirectories(t.toolkitRoot, toolkitConfigDir)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("could not create required directories: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("could not create required directories: %v", err))
	}

	err = t.installContainerLibraries(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container library: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container library: %v", err))
	}

	err = t.installContainerRuntimes(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container runtime: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container runtime: %v", err))
	}

	nvidiaContainerCliExecutable, err := t.installContainerCLI(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container CLI: %v", err))
	}

	nvidiaContainerRuntimeHookPath, err := t.installRuntimeHook(t.toolkitRoot, toolkitConfigPath)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container runtime hook: %v", err))
	}

	nvidiaCTKPath, err := t.installContainerToolkitCLI(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA Container Toolkit CLI: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA Container Toolkit CLI: %v", err))
	}

	nvidiaCDIHookPath, err := t.installContainerCDIHookCLI(t.toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA Container CDI Hook CLI: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA Container CDI Hook CLI: %v", err))
	}

	err = t.installToolkitConfig(cli, toolkitConfigPath, nvidiaContainerCliExecutable, nvidiaCTKPath, nvidiaContainerRuntimeHookPath, opts)
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

	err = t.generateCDISpec(opts, nvidiaCDIHookPath)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error generating CDI specification: %v", err)
	} else if err != nil {
		t.logger.Errorf("Ignoring error: %v", fmt.Errorf("error generating CDI specification: %v", err))
	}

	return nil
}

// installContainerLibraries locates and installs the libraries that are part of
// the nvidia-container-toolkit.
// A predefined set of library candidates are considered, with the first one
// resulting in success being installed to the toolkit folder. The install process
// resolves the symlink for the library and copies the versioned library itself.
func (t *Installer) installContainerLibraries(toolkitRoot string) error {
	t.logger.Infof("Installing NVIDIA container library to '%v'", toolkitRoot)

	libs := []string{
		"libnvidia-container.so.1",
		"libnvidia-container-go.so.1",
	}

	for _, l := range libs {
		err := t.installLibrary(l, toolkitRoot)
		if err != nil {
			return fmt.Errorf("failed to install %s: %v", l, err)
		}
	}

	return nil
}

// installLibrary installs the specified library to the toolkit directory.
func (t *Installer) installLibrary(libName string, toolkitRoot string) error {
	libraryPath, err := t.findLibrary(libName)
	if err != nil {
		return fmt.Errorf("error locating NVIDIA container library: %v", err)
	}

	installedLibPath, err := t.installFileToFolder(toolkitRoot, libraryPath)
	if err != nil {
		return fmt.Errorf("error installing %v to %v: %v", libraryPath, toolkitRoot, err)
	}
	t.logger.Infof("Installed '%v' to '%v'", libraryPath, installedLibPath)

	if filepath.Base(installedLibPath) == libName {
		return nil
	}

	err = t.installSymlink(toolkitRoot, libName, installedLibPath)
	if err != nil {
		return fmt.Errorf("error installing symlink for NVIDIA container library: %v", err)
	}

	return nil
}

// installToolkitConfig installs the config file for the NVIDIA container toolkit ensuring
// that the settings are updated to match the desired install and nvidia driver directories.
func (t *Installer) installToolkitConfig(c *cli.Context, toolkitConfigPath string, nvidiaContainerCliExecutablePath string, nvidiaCTKPath string, nvidaContainerRuntimeHookPath string, opts *Options) error {
	t.logger.Infof("Installing NVIDIA container toolkit config '%v'", toolkitConfigPath)

	cfg, err := config.New(
		config.WithConfigFile(nvidiaContainerToolkitConfigSource),
	)
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
		"nvidia-container-runtime-hook.path":                nvidaContainerRuntimeHookPath,
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

// installContainerToolkitCLI installs the nvidia-ctk CLI executable and wrapper.
func (t *Installer) installContainerToolkitCLI(toolkitDir string) (string, error) {
	e := executable{
		fileInstaller: t.fileInstaller,
		source:        "/usr/bin/nvidia-ctk",
		target: executableTarget{
			dotfileName: "nvidia-ctk.real",
			wrapperName: "nvidia-ctk",
		},
	}

	return e.install(toolkitDir)
}

// installContainerCDIHookCLI installs the nvidia-cdi-hook CLI executable and wrapper.
func (t *Installer) installContainerCDIHookCLI(toolkitDir string) (string, error) {
	e := executable{
		fileInstaller: t.fileInstaller,
		source:        "/usr/bin/nvidia-cdi-hook",
		target: executableTarget{
			dotfileName: "nvidia-cdi-hook.real",
			wrapperName: "nvidia-cdi-hook",
		},
	}

	return e.install(toolkitDir)
}

// installContainerCLI sets up the NVIDIA container CLI executable, copying the executable
// and implementing the required wrapper
func (t *Installer) installContainerCLI(toolkitRoot string) (string, error) {
	t.logger.Infof("Installing NVIDIA container CLI from '%v'", nvidiaContainerCliSource)

	env := map[string]string{
		"LD_LIBRARY_PATH": toolkitRoot,
	}

	e := executable{
		fileInstaller: t.fileInstaller,
		source:        nvidiaContainerCliSource,
		target: executableTarget{
			dotfileName: "nvidia-container-cli.real",
			wrapperName: "nvidia-container-cli",
		},
		env: env,
	}

	installedPath, err := e.install(toolkitRoot)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	}
	return installedPath, nil
}

// installRuntimeHook sets up the NVIDIA runtime hook, copying the executable
// and implementing the required wrapper
func (t *Installer) installRuntimeHook(toolkitRoot string, configFilePath string) (string, error) {
	t.logger.Infof("Installing NVIDIA container runtime hook from '%v'", nvidiaContainerRuntimeHookSource)

	argLines := []string{
		fmt.Sprintf("-config \"%s\"", configFilePath),
	}

	e := executable{
		fileInstaller: t.fileInstaller,
		source:        nvidiaContainerRuntimeHookSource,
		target: executableTarget{
			dotfileName: "nvidia-container-runtime-hook.real",
			wrapperName: "nvidia-container-runtime-hook",
		},
		argLines: argLines,
	}

	installedPath, err := e.install(toolkitRoot)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	}

	err = t.installSymlink(toolkitRoot, "nvidia-container-toolkit", installedPath)
	if err != nil {
		return "", fmt.Errorf("error installing symlink to NVIDIA container runtime hook: %v", err)
	}

	return installedPath, nil
}

// installSymlink creates a symlink in the toolkitDirectory that points to the specified target.
// Note: The target is assumed to be local to the toolkit directory
func (t *Installer) installSymlink(toolkitRoot string, link string, target string) error {
	symlinkPath := filepath.Join(toolkitRoot, link)
	targetPath := filepath.Base(target)
	t.logger.Infof("Creating symlink '%v' -> '%v'", symlinkPath, targetPath)

	err := os.Symlink(targetPath, symlinkPath)
	if err != nil {
		return fmt.Errorf("error creating symlink '%v' => '%v': %v", symlinkPath, targetPath, err)
	}
	return nil
}

// findLibrary searches a set of candidate libraries in the specified root for
// a given library name
func (t *Installer) findLibrary(libName string) (string, error) {
	t.logger.Infof("Finding library %v (root=%v)", libName)

	candidateDirs := []string{
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
	}

	for _, d := range candidateDirs {
		l := filepath.Join(t.sourceRoot, d, libName)
		t.logger.Infof("Checking library candidate '%v'", l)

		libraryCandidate, err := t.resolveLink(l)
		if err != nil {
			t.logger.Infof("Skipping library candidate '%v': %v", l, err)
			continue
		}

		return strings.TrimPrefix(libraryCandidate, t.sourceRoot), nil
	}

	return "", fmt.Errorf("error locating library '%v'", libName)
}

// resolveLink finds the target of a symlink or the file itself in the
// case of a regular file.
// This is equivalent to running `readlink -f ${l}`
func (t *Installer) resolveLink(l string) (string, error) {
	resolved, err := filepath.EvalSymlinks(l)
	if err != nil {
		return "", fmt.Errorf("error resolving link '%v': %v", l, err)
	}
	if l != resolved {
		t.logger.Infof("Resolved link: '%v' => '%v'", l, resolved)
	}
	return resolved, nil
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
