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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
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

	toolkitPidFilename = "toolkit.pid"
)

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

	cdiEnabled   bool
	cdiOutputDir string
	cdiKind      string
	cdiVendor    string
	cdiClass     string

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
			Destination: &opts.cdiEnabled,
			EnvVars:     []string{"CDI_ENABLED", "ENABLE_CDI"},
		},
		&cli.StringFlag{
			Name:        "cdi-output-dir",
			Usage:       "the directory where the CDI output files are to be written. If this is set to '', no CDI specification is generated.",
			Value:       "/var/run/cdi",
			Destination: &opts.cdiOutputDir,
			EnvVars:     []string{"CDI_OUTPUT_DIR"},
		},
		&cli.StringFlag{
			Name:        "cdi-kind",
			Usage:       "the vendor string to use for the generated CDI specification",
			Value:       "management.nvidia.com/gpu",
			Destination: &opts.cdiKind,
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

// ValidateOptions checks whether the specified options are valid
func ValidateOptions(opts *Options, toolkitRoot string) error {
	if toolkitRoot == "" {
		return fmt.Errorf("invalid --toolkit-root option: %v", toolkitRoot)
	}

	vendor, class := parser.ParseQualifier(opts.cdiKind)
	if err := parser.ValidateVendorName(vendor); err != nil {
		return fmt.Errorf("invalid CDI vendor name: %v", err)
	}
	if err := parser.ValidateClassName(class); err != nil {
		return fmt.Errorf("invalid CDI class name: %v", err)
	}
	opts.cdiVendor = vendor
	opts.cdiClass = class

	if opts.cdiEnabled && opts.cdiOutputDir == "" {
		log.Warning("Skipping CDI spec generation (no output directory specified)")
		opts.cdiEnabled = false
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
	if !opts.cdiEnabled && !isDisabled {
		log.Info("disabling device node creation since --cdi-enabled=false")
		isDisabled = true
	}
	if isDisabled {
		opts.createDeviceNodes = *cli.NewStringSlice()
	}

	return nil
}

// TryDelete attempts to remove the specified toolkit folder.
// A toolkit.pid file -- if present -- is skipped.
func TryDelete(cli *cli.Context, toolkitRoot string) error {
	log.Infof("Attempting to delete NVIDIA container toolkit from '%v'", toolkitRoot)

	contents, err := os.ReadDir(toolkitRoot)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read the contents of %v: %w", toolkitRoot, err)
	}

	for _, content := range contents {
		if content.Name() == toolkitPidFilename {
			continue
		}
		name := filepath.Join(toolkitRoot, content.Name())
		if err := os.RemoveAll(name); err != nil {
			log.Warningf("could not remove %v: %v", name, err)
		}
	}
	if err := os.RemoveAll(toolkitRoot); err != nil {
		log.Warningf("could not remove %v: %v", toolkitRoot, err)
	}
	return nil
}

// Install installs the components of the NVIDIA container toolkit.
// The specified sourceRoot is searched for the components to install.
// Any existing installation is removed.
func Install(cli *cli.Context, opts *Options, sourceRoot string, toolkitRoot string) error {
	log.Infof("Installing NVIDIA container toolkit to '%v'", toolkitRoot)

	log.Infof("Removing existing NVIDIA container toolkit installation")
	err := os.RemoveAll(toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error removing toolkit directory: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error removing toolkit directory: %v", err))
	}

	toolkitConfigDir := filepath.Join(toolkitRoot, ".config", "nvidia-container-runtime")
	toolkitConfigPath := filepath.Join(toolkitConfigDir, configFilename)

	err = createDirectories(toolkitRoot, toolkitConfigDir)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("could not create required directories: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("could not create required directories: %v", err))
	}

	err = installContainerLibraries(sourceRoot, toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container library: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container library: %v", err))
	}

	err = installContainerRuntimes(sourceRoot, toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container runtime: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container runtime: %v", err))
	}

	nvidiaContainerCliExecutable, err := installContainerCLI(sourceRoot, toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container CLI: %v", err))
	}

	nvidiaContainerRuntimeHookPath, err := installRuntimeHook(sourceRoot, toolkitRoot, toolkitConfigPath)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container runtime hook: %v", err))
	}

	nvidiaCTKPath, err := installContainerToolkitCLI(sourceRoot, toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA Container Toolkit CLI: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA Container Toolkit CLI: %v", err))
	}

	nvidiaCDIHookPath, err := installContainerCDIHookCLI(sourceRoot, toolkitRoot)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA Container CDI Hook CLI: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA Container CDI Hook CLI: %v", err))
	}

	err = installToolkitConfig(cli, toolkitConfigPath, nvidiaContainerCliExecutable, nvidiaCTKPath, nvidiaContainerRuntimeHookPath, opts)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error installing NVIDIA container toolkit config: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error installing NVIDIA container toolkit config: %v", err))
	}

	err = createDeviceNodes(opts)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error creating device nodes: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error creating device nodes: %v", err))
	}

	err = generateCDISpec(opts, nvidiaCDIHookPath)
	if err != nil && !opts.ignoreErrors {
		return fmt.Errorf("error generating CDI specification: %v", err)
	} else if err != nil {
		log.Errorf("Ignoring error: %v", fmt.Errorf("error generating CDI specification: %v", err))
	}

	return nil
}

// installContainerLibraries locates and installs the libraries that are part of
// the nvidia-container-toolkit.
// A predefined set of library candidates are considered, with the first one
// resulting in success being installed to the toolkit folder. The install process
// resolves the symlink for the library and copies the versioned library itself.
func installContainerLibraries(sourceRoot string, toolkitRoot string) error {
	log.Infof("Installing NVIDIA container library to '%v'", toolkitRoot)

	libs := []string{
		"libnvidia-container.so.1",
		"libnvidia-container-go.so.1",
	}

	for _, l := range libs {
		err := installLibrary(l, sourceRoot, toolkitRoot)
		if err != nil {
			return fmt.Errorf("failed to install %s: %v", l, err)
		}
	}

	return nil
}

// installLibrary installs the specified library to the toolkit directory.
func installLibrary(libName string, sourceRoot string, toolkitRoot string) error {
	libraryPath, err := findLibrary(sourceRoot, libName)
	if err != nil {
		return fmt.Errorf("error locating NVIDIA container library: %v", err)
	}

	installedLibPath, err := installFileToFolder(toolkitRoot, libraryPath)
	if err != nil {
		return fmt.Errorf("error installing %v to %v: %v", libraryPath, toolkitRoot, err)
	}
	log.Infof("Installed '%v' to '%v'", libraryPath, installedLibPath)

	if filepath.Base(installedLibPath) == libName {
		return nil
	}

	err = installSymlink(toolkitRoot, libName, installedLibPath)
	if err != nil {
		return fmt.Errorf("error installing symlink for NVIDIA container library: %v", err)
	}

	return nil
}

// installToolkitConfig installs the config file for the NVIDIA container toolkit ensuring
// that the settings are updated to match the desired install and nvidia driver directories.
func installToolkitConfig(c *cli.Context, toolkitConfigPath string, nvidiaContainerCliExecutablePath string, nvidiaCTKPath string, nvidaContainerRuntimeHookPath string, opts *Options) error {
	log.Infof("Installing NVIDIA container toolkit config '%v'", toolkitConfigPath)

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
			log.Infof("Skipping unset option: %v", key)
			continue
		}
		if value == nil {
			log.Infof("Skipping option with nil value: %v", key)
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
			log.Warningf("Unexpected type for option %v=%v: %T", key, value, v)
		}

		cfg.Set(key, value)
	}

	if _, err := cfg.WriteTo(targetConfig); err != nil {
		return fmt.Errorf("error writing config: %v", err)
	}

	os.Stdout.WriteString("Using config:\n")
	if _, err = cfg.WriteTo(os.Stdout); err != nil {
		log.Warningf("Failed to output config to STDOUT: %v", err)
	}

	return nil
}

// installContainerToolkitCLI installs the nvidia-ctk CLI executable and wrapper.
func installContainerToolkitCLI(sourceRoot string, toolkitDir string) (string, error) {
	e := executable{
		source: filepath.Join(sourceRoot, "/usr/bin/nvidia-ctk"),
		target: executableTarget{
			wrapperName: "nvidia-ctk",
		},
	}

	return e.install(toolkitDir)
}

// installContainerCDIHookCLI installs the nvidia-cdi-hook CLI executable and wrapper.
func installContainerCDIHookCLI(sourceRoot string, toolkitDir string) (string, error) {
	e := executable{
		source: filepath.Join(sourceRoot, "/usr/bin/nvidia-cdi-hook"),
		target: executableTarget{
			wrapperName: "nvidia-cdi-hook",
		},
	}

	return e.install(toolkitDir)
}

// installContainerCLI sets up the NVIDIA container CLI executable, copying the executable
// and implementing the required wrapper
func installContainerCLI(sourceRoot string, toolkitRoot string) (string, error) {
	log.Infof("Installing NVIDIA container CLI from '%v'", nvidiaContainerCliSource)

	envm := map[string]string{
		"LD_LIBRARY_PATH": toolkitRoot,
	}

	e := executable{
		source: filepath.Join(sourceRoot, nvidiaContainerCliSource),
		target: executableTarget{
			wrapperName: "nvidia-container-cli",
		},
		envm: envm,
	}

	installedPath, err := e.install(toolkitRoot)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	}
	return installedPath, nil
}

// installRuntimeHook sets up the NVIDIA runtime hook, copying the executable
// and implementing the required wrapper
func installRuntimeHook(sourceRoot string, toolkitRoot string, configFilePath string) (string, error) {
	log.Infof("Installing NVIDIA container runtime hook from '%v'", nvidiaContainerRuntimeHookSource)

	e := executable{
		source: filepath.Join(sourceRoot, nvidiaContainerRuntimeHookSource),
		target: executableTarget{
			wrapperName: "nvidia-container-runtime-hook",
		},
		argv: []string{"-config", configFilePath},
	}

	installedPath, err := e.install(toolkitRoot)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	}

	err = installSymlink(toolkitRoot, "nvidia-container-toolkit", installedPath)
	if err != nil {
		return "", fmt.Errorf("error installing symlink to NVIDIA container runtime hook: %v", err)
	}

	return installedPath, nil
}

// installSymlink creates a symlink in the toolkitDirectory that points to the specified target.
// Note: The target is assumed to be local to the toolkit directory
func installSymlink(toolkitRoot string, link string, target string) error {
	symlinkPath := filepath.Join(toolkitRoot, link)
	targetPath := filepath.Base(target)
	log.Infof("Creating symlink '%v' -> '%v'", symlinkPath, targetPath)

	err := os.Symlink(targetPath, symlinkPath)
	if err != nil {
		return fmt.Errorf("error creating symlink '%v' => '%v': %v", symlinkPath, targetPath, err)
	}
	return nil
}

// installFileToFolder copies a source file to a destination folder.
// The path of the input file is ignored.
// e.g. installFileToFolder("/some/path/file.txt", "/output/path")
// will result in a file "/output/path/file.txt" being generated
func installFileToFolder(destFolder string, src string) (string, error) {
	name := filepath.Base(src)
	return installFileToFolderWithName(destFolder, name, src)
}

// cp src destFolder/name
func installFileToFolderWithName(destFolder string, name, src string) (string, error) {
	dest := filepath.Join(destFolder, name)
	err := installFile(dest, src)
	if err != nil {
		return "", fmt.Errorf("error copying '%v' to '%v': %v", src, dest, err)
	}
	return dest, nil
}

// installFile copies a file from src to dest and maintains
// file modes
func installFile(dest string, src string) error {
	log.Infof("Installing '%v' to '%v'", src, dest)

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source: %v", err)
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating destination: %v", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	err = applyModeFromSource(dest, src)
	if err != nil {
		return fmt.Errorf("error setting destination file mode: %v", err)
	}
	return nil
}

// applyModeFromSource sets the file mode for a destination file
// to match that of a specified source file
func applyModeFromSource(dest string, src string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting file info for '%v': %v", src, err)
	}
	err = os.Chmod(dest, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("error setting mode for '%v': %v", dest, err)
	}
	return nil
}

// findLibrary searches a set of candidate libraries in the specified root for
// a given library name
func findLibrary(root string, libName string) (string, error) {
	log.Infof("Finding library %v (root=%v)", libName, root)

	candidateDirs := []string{
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
	}

	for _, d := range candidateDirs {
		l := filepath.Join(root, d, libName)
		log.Infof("Checking library candidate '%v'", l)

		libraryCandidate, err := resolveLink(l)
		if err != nil {
			log.Infof("Skipping library candidate '%v': %v", l, err)
			continue
		}

		return libraryCandidate, nil
	}

	return "", fmt.Errorf("error locating library '%v'", libName)
}

// resolveLink finds the target of a symlink or the file itself in the
// case of a regular file.
// This is equivalent to running `readlink -f ${l}`
func resolveLink(l string) (string, error) {
	resolved, err := filepath.EvalSymlinks(l)
	if err != nil {
		return "", fmt.Errorf("error resolving link '%v': %v", l, err)
	}
	if l != resolved {
		log.Infof("Resolved link: '%v' => '%v'", l, resolved)
	}
	return resolved, nil
}

func createDirectories(dir ...string) error {
	for _, d := range dir {
		log.Infof("Creating directory '%v'", d)
		err := os.MkdirAll(d, 0755)
		if err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}
	return nil
}

func createDeviceNodes(opts *Options) error {
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
		log.Infof("Creating %v device nodes at %v", mode, opts.DevRootCtrPath)
		if mode != "control" {
			log.Warningf("Unrecognised device mode: %v", mode)
			continue
		}
		if err := devices.CreateNVIDIAControlDevices(); err != nil {
			return fmt.Errorf("failed to create control device nodes: %v", err)
		}
	}
	return nil
}

// generateCDISpec generates a CDI spec for use in management containers
func generateCDISpec(opts *Options, nvidiaCDIHookPath string) error {
	if !opts.cdiEnabled {
		return nil
	}
	log.Info("Generating CDI spec for management containers")
	cdilib, err := nvcdi.New(
		nvcdi.WithMode(nvcdi.ModeManagement),
		nvcdi.WithDriverRoot(opts.DriverRootCtrPath),
		nvcdi.WithDevRoot(opts.DevRootCtrPath),
		nvcdi.WithNVIDIACDIHookPath(nvidiaCDIHookPath),
		nvcdi.WithVendor(opts.cdiVendor),
		nvcdi.WithClass(opts.cdiClass),
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
	err = spec.Save(filepath.Join(opts.cdiOutputDir, name))
	if err != nil {
		return fmt.Errorf("failed to save CDI spec for management containers: %v", err)
	}

	return nil
}
