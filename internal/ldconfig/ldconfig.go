/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package ldconfig

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

const (
	// ldsoconfdFilenamePattern specifies the pattern for the filename
	// in ld.so.conf.d that includes references to the specified directories.
	// The 00-nvcr prefix is chosen to ensure that these libraries have a
	// higher precedence than other libraries on the system, but lower than
	// the 00-cuda-compat that is included in some containers.
	ldsoconfdFilenamePattern = "00-nvcr-*.conf"
)

type Ldconfig struct {
	ldconfigPath          string
	inRoot                string
	isDebianLikeHost      bool
	isDebianLikeContainer bool
	directories           []string
}

// NewRunner creates an exec.Cmd that can be used to run ldconfig.
func NewRunner(id string, ldconfigPath string, containerRoot string, additionalargs ...string) (*exec.Cmd, error) {
	args := []string{
		id,
		"--ldconfig-path", strings.TrimPrefix(config.NormalizeLDConfigPath("@"+ldconfigPath), "@"),
		"--container-root", containerRoot,
	}
	if isDebian() {
		args = append(args, "--is-debian-like-host")
	}
	args = append(args, additionalargs...)

	return createReexecCommand(args)
}

// NewFromArgs creates an Ldconfig struct from the args passed to the Cmd
// above.
// This struct is used to perform operations on the ldcache and libraries in a
// particular root (e.g. a container).
//
// args[0] is the reexec initializer function name
// The following flags are required:
//
//	--ldconfig-path=LDCONFIG_PATH	the path to ldconfig on the host
//	--container-root=CONTAINER_ROOT	the path in which ldconfig must be run
//
// The following flags are optional:
//
//	--is-debian-like-host	Indicates that the host system is debian-based.
//
// The remaining args are folders where soname symlinks need to be created.
func NewFromArgs(args ...string) (*Ldconfig, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("incorrect arguments: %v", args)
	}
	fs := flag.NewFlagSet(args[1], flag.ExitOnError)
	ldconfigPath := fs.String("ldconfig-path", "", "the path to ldconfig on the host")
	containerRoot := fs.String("container-root", "", "the path in which ldconfig must be run")
	isDebianLikeHost := fs.Bool("is-debian-like-host", false, "the hook is running from a Debian-like host")
	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}

	if *ldconfigPath == "" {
		return nil, fmt.Errorf("an ldconfig path must be specified")
	}
	if *containerRoot == "" || *containerRoot == "/" {
		return nil, fmt.Errorf("ldconfig must be run in the non-system root")
	}

	l := &Ldconfig{
		ldconfigPath:          *ldconfigPath,
		inRoot:                *containerRoot,
		isDebianLikeHost:      *isDebianLikeHost,
		isDebianLikeContainer: isDebian(),
		directories:           fs.Args(),
	}
	return l, nil
}

func (l *Ldconfig) UpdateLDCache() error {
	ldconfigPath, err := l.prepareRoot()
	if err != nil {
		return err
	}

	// Explicitly specify using /etc/ld.so.conf since the host's ldconfig may
	// be configured to use a different config file by default.
	const topLevelLdsoconfFilePath = "/etc/ld.so.conf"
	filteredDirectories, err := l.filterDirectories(topLevelLdsoconfFilePath, l.directories...)
	if err != nil {
		return err
	}

	args := []string{
		filepath.Base(ldconfigPath),
		"-f", topLevelLdsoconfFilePath,
		"-C", "/etc/ld.so.cache",
	}

	if err := createLdsoconfdFile(ldsoconfdFilenamePattern, filteredDirectories...); err != nil {
		return fmt.Errorf("failed to update ld.so.conf.d: %w", err)
	}

	return SafeExec(ldconfigPath, args, nil)
}

func (l *Ldconfig) prepareRoot() (string, error) {
	root, err := os.OpenRoot(l.inRoot)
	if err != nil {
		return "", fmt.Errorf("failed to open root: %w", err)
	}
	defer root.Close()

	// To prevent leaking the parent proc filesystem, we create a new proc mount
	// in the specified root.
	if err := mountProc(root); err != nil {
		return "", fmt.Errorf("error mounting /proc: %w", err)
	}

	// We mount the host ldconfig before we pivot root since host paths are not
	// visible after the pivot root operation.
	ldconfigPath, err := mountLdConfig(l.ldconfigPath, root)
	if err != nil {
		return "", fmt.Errorf("error mounting host ldconfig: %w", err)
	}

	// We pivot to the container root for the new process, this further limits
	// access to the host.
	if err := pivotRoot(root.Name()); err != nil {
		return "", fmt.Errorf("error running pivot_root: %w", err)
	}

	return ldconfigPath, nil
}

func (l *Ldconfig) filterDirectories(configFilePath string, directories ...string) ([]string, error) {
	ldconfigDirs, err := l.getLdsoconfDirectories(configFilePath)
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, d := range directories {
		if _, ok := ldconfigDirs[d]; ok {
			continue
		}
		filtered = append(filtered, d)
	}
	return filtered, nil
}

// createLdsoconfdFile creates a file at /etc/ld.so.conf.d/.
// The file is created at /etc/ld.so.conf.d/{{ .pattern }} using `CreateTemp` and
// contains the specified directories on each line.
func createLdsoconfdFile(pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		return nil
	}

	ldsoconfdDir := "/etc/ld.so.conf.d"
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		_ = configFile.Close()
	}()

	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err = fmt.Fprintf(configFile, "%s\n", dir)
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}

// getLdsoconfDirectories returns a map of ldsoconf directories to the conf
// files that refer to the directory.
func (l *Ldconfig) getLdsoconfDirectories(configFilePath string) (map[string]struct{}, error) {
	ldconfigDirs := make(map[string]struct{})
	for _, d := range l.getSystemSerachPaths() {
		ldconfigDirs[d] = struct{}{}
	}

	processedConfFiles := make(map[string]bool)
	ldsoconfFilenames := []string{configFilePath}
	for len(ldsoconfFilenames) > 0 {
		ldsoconfFilename := ldsoconfFilenames[0]
		ldsoconfFilenames = ldsoconfFilenames[1:]
		if processedConfFiles[ldsoconfFilename] {
			continue
		}
		processedConfFiles[ldsoconfFilename] = true

		if len(ldsoconfFilename) == 0 {
			continue
		}
		directories, includedFilenames, err := processLdsoconfFile(ldsoconfFilename)
		if err != nil {
			return nil, err
		}
		ldsoconfFilenames = append(ldsoconfFilenames, includedFilenames...)
		for _, d := range directories {
			ldconfigDirs[d] = struct{}{}
		}
	}
	return ldconfigDirs, nil
}

func (l *Ldconfig) getSystemSerachPaths() []string {
	if l.isDebianLikeContainer {
		debianSystemSearchPaths()
	}
	return nonDebianSystemSearchPaths()
}

// processLdsoconfFile extracts the list of directories and included configs
// from the specified file.
func processLdsoconfFile(ldsoconfFilename string) ([]string, []string, error) {
	ldsoconf, err := os.Open(ldsoconfFilename)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	defer ldsoconf.Close()

	var directories []string
	var includedFilenames []string
	scanner := bufio.NewScanner(ldsoconf)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "#") || len(line) == 0:
			continue
		case strings.HasPrefix(line, "include "):
			include, err := filepath.Glob(strings.TrimPrefix(line, "include "))
			if err != nil {
				// We ignore invalid includes.
				// TODO: How does ldconfig handle this?
				continue
			}
			includedFilenames = append(includedFilenames, include...)
		default:
			directories = append(directories, line)
		}
	}
	return directories, includedFilenames, nil
}

func isDebian() bool {
	info, err := os.Stat("/etc/debian_version")
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// nonDebianSystemSearchPaths returns the system search paths for non-Debian
// systems.
//
// This list was taken from the output of:
//
//	docker run --rm -ti redhat/ubi9 /usr/lib/ld-linux-aarch64.so.1 --help | grep -A6 "Shared library search path"
func nonDebianSystemSearchPaths() []string {
	return []string{"/lib64", "/usr/lib64"}
}

// debianSystemSearchPaths returns the system search paths for Debian-like
// systems.
//
// This list was taken from the output of:
//
//	docker run --rm -ti ubuntu /usr/lib/aarch64-linux-gnu/ld-linux-aarch64.so.1 --help | grep -A6 "Shared library search path"
func debianSystemSearchPaths() []string {
	var paths []string
	switch runtime.GOARCH {
	case "amd64":
		paths = append(paths,
			"/lib/x86_64-linux-gnu",
			"/usr/lib/x86_64-linux-gnu",
		)
	case "arm64":
		paths = append(paths,
			"/lib/aarch64-linux-gnu",
			"/usr/lib/aarch64-linux-gnu",
		)
	}
	paths = append(paths, "/lib", "/usr/lib")

	return paths
}
