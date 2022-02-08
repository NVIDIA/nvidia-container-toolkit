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

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	// DefaultNvidiaDriverRoot specifies the default NVIDIA driver run directory
	DefaultNvidiaDriverRoot = "/run/nvidia/driver"

	nvidiaContainerCliSource         = "/usr/bin/nvidia-container-cli"
	nvidiaContainerRuntimeHookSource = "/usr/bin/nvidia-container-toolkit"

	nvidiaContainerToolkitConfigSource = "/etc/nvidia-container-runtime/config.toml"
	configFilename                     = "config.toml"
)

var toolkitDirArg string
var nvidiaDriverRootFlag string
var nvidiaContainerRuntimeDebugFlag string
var nvidiaContainerRuntimeLogLevelFlag string
var nvidiaContainerCLIDebugFlag string

func main() {
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "toolkit"
	c.Usage = "Manage the NVIDIA container toolkit"
	c.Version = "0.1.0"

	// Create the 'install' subcommand
	install := cli.Command{}
	install.Name = "install"
	install.Usage = "Install the components of the NVIDIA container toolkit"
	install.ArgsUsage = "<toolkit_directory>"
	install.Before = parseArgs
	install.Action = Install

	// Create the 'delete' command
	delete := cli.Command{}
	delete.Name = "delete"
	delete.Usage = "Delete the NVIDIA container toolkit"
	delete.ArgsUsage = "<toolkit_directory>"
	delete.Before = parseArgs
	delete.Action = Delete

	// Register the subcommand with the top-level CLI
	c.Commands = []*cli.Command{
		&install,
		&delete,
	}

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "nvidia-driver-root",
			Value:       DefaultNvidiaDriverRoot,
			Destination: &nvidiaDriverRootFlag,
			EnvVars:     []string{"NVIDIA_DRIVER_ROOT"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime-debug",
			Usage:       "Specify the location of the debug log file for the NVIDIA Container Runtime",
			Destination: &nvidiaContainerRuntimeDebugFlag,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_DEBUG"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-runtime-debug-log-level",
			Destination: &nvidiaContainerRuntimeLogLevelFlag,
			EnvVars:     []string{"NVIDIA_CONTAINER_RUNTIME_LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:        "nvidia-container-cli-debug",
			Usage:       "Specify the location of the debug log file for the NVIDIA Container CLI",
			Destination: &nvidiaContainerCLIDebugFlag,
			EnvVars:     []string{"NVIDIA_CONTAINER_CLI_DEBUG"},
		},
	}

	// Update the subcommand flags with the common subcommand flags
	install.Flags = append([]cli.Flag{}, flags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Fatal(fmt.Errorf("error: %v", err))
	}
}

// parseArgs parses the command line arguments to the CLI
func parseArgs(c *cli.Context) error {
	args := c.Args()

	log.Infof("Parsing arguments: %v", args.Slice())
	if c.NArg() != 1 {
		return fmt.Errorf("incorrect number of arguments")
	}
	toolkitDirArg = args.Get(0)
	log.Infof("Successfully parsed arguments")

	return nil
}

// Delete removes the NVIDIA container toolkit
func Delete(cli *cli.Context) error {
	log.Infof("Deleting NVIDIA container toolkit from '%v'", toolkitDirArg)
	err := os.RemoveAll(toolkitDirArg)
	if err != nil {
		return fmt.Errorf("error deleting toolkit directory: %v", err)
	}
	return nil
}

// Install installs the components of the NVIDIA container toolkit.
// Any existing installation is removed.
func Install(cli *cli.Context) error {
	log.Infof("Installing NVIDIA container toolkit to '%v'", toolkitDirArg)

	log.Infof("Removing existing NVIDIA container toolkit installation")
	err := os.RemoveAll(toolkitDirArg)
	if err != nil {
		return fmt.Errorf("error removing toolkit directory: %v", err)
	}

	toolkitConfigDir := filepath.Join(toolkitDirArg, ".config", "nvidia-container-runtime")
	toolkitConfigPath := filepath.Join(toolkitConfigDir, configFilename)

	err = createDirectories(toolkitDirArg, toolkitConfigDir)
	if err != nil {
		return fmt.Errorf("could not create required directories: %v", err)
	}

	err = installContainerLibraries(toolkitDirArg)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container library: %v", err)
	}

	err = installContainerRuntimes(toolkitDirArg, nvidiaDriverRootFlag)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container runtime: %v", err)
	}

	nvidiaContainerCliExecutable, err := installContainerCLI(toolkitDirArg)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	}

	_, err = installRuntimeHook(toolkitDirArg, toolkitConfigPath)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	}

	err = installToolkitConfig(toolkitConfigPath, nvidiaDriverRootFlag, nvidiaContainerCliExecutable)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container toolkit config: %v", err)
	}

	return nil
}

// installContainerLibraries locates and installs the libraries that are part of
// the nvidia-container-toolkit.
// A predefined set of library candidates are considered, with the first one
// resulting in success being installed to the toolkit folder. The install process
// resolves the symlink for the library and copies the versioned library itself.
func installContainerLibraries(toolkitDir string) error {
	log.Infof("Installing NVIDIA container library to '%v'", toolkitDir)

	libs := []string{
		"libnvidia-container.so.1",
		"libnvidia-container-go.so.1",
	}

	for _, l := range libs {
		err := installLibrary(l, toolkitDir)
		if err != nil {
			return fmt.Errorf("failed to install %s: %v", l, err)
		}
	}

	return nil
}

// installLibrary installs the specified library to the toolkit directory.
func installLibrary(libName string, toolkitDir string) error {
	libraryPath, err := findLibrary("", libName)
	if err != nil {
		return fmt.Errorf("error locating NVIDIA container library: %v", err)
	}

	installedLibPath, err := installFileToFolder(toolkitDir, libraryPath)
	if err != nil {
		return fmt.Errorf("error installing %v to %v: %v", libraryPath, toolkitDir, err)
	}
	log.Infof("Installed '%v' to '%v'", libraryPath, installedLibPath)

	if filepath.Base(installedLibPath) == libName {
		return nil
	}

	err = installSymlink(toolkitDir, libName, installedLibPath)
	if err != nil {
		return fmt.Errorf("error installing symlink for NVIDIA container library: %v", err)
	}

	return nil
}

// installToolkitConfig installs the config file for the NVIDIA container toolkit ensuring
// that the settings are updated to match the desired install and nvidia driver directories.
func installToolkitConfig(toolkitConfigPath string, nvidiaDriverDir string, nvidiaContainerCliExecutablePath string) error {
	log.Infof("Installing NVIDIA container toolkit config '%v'", toolkitConfigPath)

	config, err := toml.LoadFile(nvidiaContainerToolkitConfigSource)
	if err != nil {
		return fmt.Errorf("could not open source config file: %v", err)
	}

	targetConfig, err := os.Create(toolkitConfigPath)
	if err != nil {
		return fmt.Errorf("could not create target config file: %v", err)
	}
	defer targetConfig.Close()

	nvidiaContainerCliKey := func(p string) []string {
		return []string{"nvidia-container-cli", p}
	}

	// Read the ldconfig path from the config as this may differ per platform
	// On ubuntu-based systems this ends in `.real`
	ldconfigPath := fmt.Sprintf("%s", config.GetPath(nvidiaContainerCliKey("ldconfig")))

	// Use the driver run root as the root:
	driverLdconfigPath := "@" + filepath.Join(nvidiaDriverDir, strings.TrimPrefix(ldconfigPath, "@/"))

	config.SetPath(nvidiaContainerCliKey("root"), nvidiaDriverDir)
	config.SetPath(nvidiaContainerCliKey("path"), nvidiaContainerCliExecutablePath)
	config.SetPath(nvidiaContainerCliKey("ldconfig"), driverLdconfigPath)

	// Set the debug options if selected
	debugOptions := map[string]string{
		"nvidia-container-runtime.debug":     nvidiaContainerRuntimeDebugFlag,
		"nvidia-container-runtime.log-level": nvidiaContainerRuntimeLogLevelFlag,
		"nvidia-container-cli.debug":         nvidiaContainerCLIDebugFlag,
	}
	for key, value := range debugOptions {
		if value == "" {
			continue
		}
		if config.Get(key) != nil {
			continue
		}
		config.Set(key, value)
	}

	_, err = config.WriteTo(targetConfig)
	if err != nil {
		return fmt.Errorf("error writing config: %v", err)
	}
	return nil
}

// installContainerCLI sets up the NVIDIA container CLI executable, copying the executable
// and implementing the required wrapper
func installContainerCLI(toolkitDir string) (string, error) {
	log.Infof("Installing NVIDIA container CLI from '%v'", nvidiaContainerCliSource)

	env := map[string]string{
		"LD_LIBRARY_PATH": toolkitDir,
	}

	e := executable{
		source: nvidiaContainerCliSource,
		target: executableTarget{
			dotfileName: "nvidia-container-cli.real",
			wrapperName: "nvidia-container-cli",
		},
		env: env,
	}

	installedPath, err := e.install(toolkitDir)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container CLI: %v", err)
	}
	return installedPath, nil
}

// installRuntimeHook sets up the NVIDIA runtime hook, copying the executable
// and implementing the required wrapper
func installRuntimeHook(toolkitDir string, configFilePath string) (string, error) {
	log.Infof("Installing NVIDIA container runtime hook from '%v'", nvidiaContainerRuntimeHookSource)

	argLines := []string{
		fmt.Sprintf("-config \"%s\"", configFilePath),
	}

	e := executable{
		source: nvidiaContainerRuntimeHookSource,
		target: executableTarget{
			dotfileName: "nvidia-container-toolkit.real",
			wrapperName: "nvidia-container-toolkit",
		},
		argLines: argLines,
	}

	installedPath, err := e.install(toolkitDir)
	if err != nil {
		return "", fmt.Errorf("error installing NVIDIA container runtime hook: %v", err)
	}

	err = installSymlink(toolkitDir, "nvidia-container-runtime-hook", installedPath)
	if err != nil {
		return "", fmt.Errorf("error installing symlink to NVIDIA container runtime hook: %v", err)
	}

	return installedPath, nil
}

// installSymlink creates a symlink in the toolkitDirectory that points to the specified target.
// Note: The target is assumed to be local to the toolkit directory
func installSymlink(toolkitDir string, link string, target string) error {
	symlinkPath := filepath.Join(toolkitDir, link)
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
