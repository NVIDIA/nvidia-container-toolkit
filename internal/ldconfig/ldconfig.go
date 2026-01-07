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
	"io"
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
	// ldsoconfdSystemDirsFilenamePattern specifies the filename pattern for the drop-in conf file
	// that includes the expected system directories for the container.
	// This is chosen to have a high likelihood of being lexicographically last in
	// in the list of config files, since system search paths should be
	// considered last.
	ldsoconfdSystemDirsFilenamePattern = "zz-nvcr-*.conf"
	// defaultTopLevelLdsoconfFilePath is the standard location of the top-level ld.so.conf file.
	// Most container images based on a distro will have this file, but distroless container images
	// may not.
	defaultTopLevelLdsoconfFilePath = "/etc/ld.so.conf"
	// defaultLdsoconfdDir is the standard location for the ld.so.conf.d drop-in directory. Most
	// container images based on a distro will have this directory included by the top-level
	// ld.so.conf file, but some may not. And some container images may not have a top-level
	// ld.so.conf file at all.
	defaultLdsoconfdDir = "/etc/ld.so.conf.d"
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
	if isDebianLike() {
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
// args[0] is the reexec initializer function name and is required.
//
// The following flags are required:
//
//	--ldconfig-path=LDCONFIG_PATH	the path to ldconfig on the host
//	--container-root=CONTAINER_ROOT	the path in which ldconfig must be run
//
// The following flags are optional:
//
//	--is-debian-like-host	Indicates that the host system is debian-like (e.g. Debian, Ubuntu)
//	                     	as opposed to non-Debian-like (e.g. RHEL, Fedora)
//	                     	See https://github.com/NVIDIA/nvidia-container-toolkit/pull/1444
//
// The remaining args are folders where soname symlinks need to be created.
func NewFromArgs(args ...string) (*Ldconfig, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("incorrect arguments: %v", args)
	}
	fs := flag.NewFlagSet("ldconfig-options", flag.ExitOnError)
	ldconfigPath := fs.String("ldconfig-path", "", "the path to ldconfig on the host")
	containerRoot := fs.String("container-root", "", "the path in which ldconfig must be run")
	isDebianLikeHost := fs.Bool("is-debian-like-host", false, `indicates that the host system is debian-based.
This allows us to handle the case where there are  differences in behavior
between the ldconfig from the host (as executed from an update-ldcache hook) and
ldconfig in the container. Such differences include system search paths.`)
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
		ldconfigPath:     *ldconfigPath,
		inRoot:           *containerRoot,
		isDebianLikeHost: *isDebianLikeHost,
		directories:      fs.Args(),
	}
	return l, nil
}

func (l *Ldconfig) UpdateLDCache() error {
	ldconfigPath, err := l.prepareRoot()
	if err != nil {
		return err
	}

	// `prepareRoot` pivots to the container root, so can now set the container "debian-ness".
	l.isDebianLikeContainer = isDebianLike()

	// Ensure that the top-level config file used specifies includes the
	// defaultLdsoconfDir drop-in config folder.
	if err := ensureLdsoconfFile(defaultTopLevelLdsoconfFilePath, defaultLdsoconfdDir); err != nil {
		return fmt.Errorf("failed to ensure ld.so.conf file: %w", err)
	}

	filteredDirectories, err := l.filterDirectories(defaultTopLevelLdsoconfFilePath, l.directories...)
	if err != nil {
		return err
	}

	args := []string{
		filepath.Base(ldconfigPath),
		"-f", defaultTopLevelLdsoconfFilePath,
		"-C", "/etc/ld.so.cache",
	}

	if err := createLdsoconfdFile(defaultLdsoconfdDir, ldsoconfdFilenamePattern, filteredDirectories...); err != nil {
		return fmt.Errorf("failed to write %s drop-in: %w", ldsoconfdFilenamePattern, err)
	}

	systemSearchPaths := l.getSystemSearchPaths()
	// In most cases, the hook will be executing a host ldconfig that may be configured widely
	// differently from what the container image expects.
	// The common case is Debian-like (e.g. Debian, Ubuntu) vs non-Debian-like (e.g. RHEL, Fedora).
	// But there are also hosts that configure ldconfig to search in a glibc prefix
	// (e.g. /usr/lib/glibc). To avoid all these cases, write the container's expected system search
	// paths to a drop-in conf file that is likely to be last in lexicographic order. Entries in the
	// top-level ld.so.conf file may be processed after this drop-in, but this hook does not modify
	// the top-level file if it exists.
	if err := createLdsoconfdFile(defaultLdsoconfdDir, ldsoconfdSystemDirsFilenamePattern, systemSearchPaths...); err != nil {
		return fmt.Errorf("failed to write %s drop-in: %w", ldsoconfdSystemDirsFilenamePattern, err)
	}

	// Also output the folders to the alpine .path file as required.
	if err := createMuslPathFileIfRequired(append(filteredDirectories, systemSearchPaths...)...); err != nil {
		return fmt.Errorf("failed to update .path file for musl: %w", err)
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
		ldconfigDirs[d] = struct{}{}
	}
	return filtered, nil
}

// createLdsoconfdFile creates a ld.so.conf.d drop-in file with the specified directories on each
// line. The file is created at `ldsoconfdDir`/{{ .pattern }} using `CreateTemp`.
func createLdsoconfdFile(ldsoconfdDir, pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		return nil
	}
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

	if err := outputListToFile(configFile, dirs...); err != nil {
		return err
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}

func outputListToFile(w io.Writer, dirs ...string) error {
	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err := fmt.Fprintf(w, "%s\n", dir)
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}
	return nil
}

// ensureLdsoconfFile creates a "standard" top-level ld.so.conf file if none exists.
//
// The created file will contain a single include statement for "`ldsoconfdDir`/*.conf".
func ensureLdsoconfFile(topLevelLdsoconfFilePath, ldsoconfdDir string) error {
	configFile, err := os.OpenFile(topLevelLdsoconfFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("failed to create top-level ld.so.conf file: %w", err)
	}
	defer configFile.Close()
	_, err = configFile.WriteString("include " + ldsoconfdDir + "/*.conf\n")
	if err != nil {
		return fmt.Errorf("failed to write to top-level ld.so.conf file: %w", err)
	}
	return nil
}

// getLdsoconfDirectories returns a map of ldsoconf directories to the conf
// files that refer to the directory.
func (l *Ldconfig) getLdsoconfDirectories(configFilePath string) (map[string]struct{}, error) {
	ldconfigDirs := make(map[string]struct{})
	for _, d := range l.getSystemSearchPaths() {
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

func (l *Ldconfig) getSystemSearchPaths() []string {
	if l.isDebianLikeContainer {
		return debianSystemSearchPaths()
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

// createMuslPathFileIfRequired creates a musl .path file that allows libraries
// from the specified directories to be discovered on the system.
// This is required because systems that use musl do not rely on the ldcache to
// discover libraries.
func createMuslPathFileIfRequired(dirs ...string) error {
	if len(dirs) == 0 || !isMusl() {
		return nil
	}

	var pathFileName string
	switch runtime.GOARCH {
	case "amd64":
		pathFileName = "/etc/ld-musl-x86_64.path"
	case "arm64":
		pathFileName = "/etc/ld-musl-aarch64.path"
	}

	pathFile, err := os.OpenFile(pathFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open .path file: %w", err)
	}
	defer func() {
		_ = pathFile.Close()
	}()

	return outputListToFile(pathFile, dirs...)
}

// isMusl checks whether the container is running musl instead of glibc.
// Note that for the time being we only check whether `/etc/alpine-release` is
// present in the container.
func isMusl() bool {
	info, err := os.Stat("/etc/alpine-release")
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// isDebianLike returns true if a Debian-like distribution is detected.
// Debian-like distributions include Debian and Ubuntu, whereas non-Debian-like
// distributions include RHEL and Fedora.
func isDebianLike() bool {
	info, err := os.Stat("/etc/debian_version")
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// nonDebianSystemSearchPaths returns the system search paths for non-Debian-like systems.
// (note that Debian-like systems include Ubuntu systems)
//
// glibc ldconfig's calls `add_system_dir` with `SLIBDIR` and `LIBDIR` (if they are not equal). On
// aarch64 and x86_64, `add_system_dir` is a macro that scans the provided path. If the path ends
// with "/lib64" (or "/libx32", x86_64 only), it strips those suffixes. Then it registers the
// resulting path. Then if the path ends with "/lib", it registers "path"+"64" (and "path"+"x32",
// x86_64 only).
//
// By default, "LIBDIR" is "/usr/lib" and "SLIBDIR" is "/lib". Note that on modern distributions,
// "/lib" is usually a symlink to "/usr/lib" and "/lib64" to "/usr/lib64". ldconfig resolves
// symlinks and skips duplicate directory entries.
//
// To get the list of system paths, you can invoke the dynamic linker with `--list-diagnostics` and
// look for "path.system_dirs". For example
//
// $ docker run --rm -ti fedora bash -c "uname -m;\$(find . | grep /ld-linux) --list-diagnostics | grep path.system_dirs"
// x86_64
// path.system_dirs[0x0]="/lib64/"
// path.system_dirs[0x1]="/usr/lib64/"
//
// $ docker run --rm -ti redhat/ubi9 bash -c "uname -m;\$(find . | grep /ld-linux) --list-diagnostics | grep path.system_dirs"
// x86_64
// path.system_dirs[0x0]="/lib64/"
// path.system_dirs[0x1]="/usr/lib64/"
//
// On most distributions, including Fedora and derivatives, this yields the following
// ldconfig system search paths.
//
// TODO: Add other architectures that have custom `add_system_dir` macros (e.g. riscv)
// TODO: Replace with executing the container's dynamlic linker with `--list-diagnostics`?
func nonDebianSystemSearchPaths() []string {
	var paths []string
	paths = append(paths, "/lib", "/usr/lib")
	switch runtime.GOARCH {
	case "amd64":
		paths = append(paths,
			"/lib64",
			"/usr/lib64",
			"/libx32",
			"/usr/libx32",
		)
	case "arm64":
		paths = append(paths,
			"/lib64",
			"/usr/lib64",
		)
	}
	return paths
}

// debianSystemSearchPaths returns the system search paths for Debian-like systems.
// (note that Debian-like systems include Ubuntu systems)
//
// Debian (and derivatives) apply their multi-arch patch to glibc, which modifies ldconfig to
// use the same set of system paths as the dynamic linker. These paths are going to include the
// multi-arch directory _and_ by default "/lib" and "/usr/lib" for compatibility.
//
// To get the list of system paths, you can invoke the dynamic linker with `--list-diagnostics` and
// look for "path.system_dirs". For example
//
// $ docker run --rm -ti ubuntu bash -c "uname -m;\$(find . | grep /ld-linux | head -1) --list-diagnostics | grep path.system_dirs"
// x86_64
// path.system_dirs[0x0]="/lib/x86_64-linux-gnu/"
// path.system_dirs[0x1]="/usr/lib/x86_64-linux-gnu/"
// path.system_dirs[0x2]="/lib/"
// path.system_dirs[0x3]="/usr/lib/"
//
// $ docker run --rm -ti debian  bash -c "uname -m;\$(find . | grep /ld-linux | head -1) --list-diagnostics | grep path.system_dirs"
// x86_64
// path.system_dirs[0x0]="/lib/x86_64-linux-gnu/"
// path.system_dirs[0x1]="/usr/lib/x86_64-linux-gnu/"
// path.system_dirs[0x2]="/lib/"
// path.system_dirs[0x3]="/usr/lib/"
//
// This yields the following ldconfig system search paths.
//
// TODO: Add other architectures that have custom `add_system_dir` macros (e.g. riscv)
// TODO: Replace with executing the container's dynamlic linker with `--list-diagnostics`?
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
