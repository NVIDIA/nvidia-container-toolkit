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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	ldconfigPath string
	inRoot       string
	isRootless   bool
}

// NewRunner creates an exec.Cmd that can be used to run ldconfig.
func NewRunner(id string, ldconfigPath string, containerRoot string, additionalargs ...string) (*exec.Cmd, error) {
	args := []string{
		id,
		strings.TrimPrefix(config.NormalizeLDConfigPath("@"+ldconfigPath), "@"),
		containerRoot,
		fmt.Sprintf("rootless=%v", os.Geteuid() != 0),
	}
	args = append(args, additionalargs...)

	return createReexecCommand(args)
}

// NewFromRunnerArgs creates an Ldconfig struct from the args passed to the Cmd
// above.
// This struct is used to perform operations on the ldcache and libraries in a
// particular root (e.g. a container).
//
// args[0] is the reexec initializer function name
// args[1] is the path of the ldconfig binary on the host
// args[2] is the container root directory
// args[3] is the optional flag indicating whether the container is being run rootless.
// The remaining args are folders where soname symlinks need to be created.
func NewFromArgs(args ...string) (*Ldconfig, []string, error) {
	// Validate the number of arguments: [program, ldconfigPath, inRoot]
	if len(args) < 4 {
		return nil, args, fmt.Errorf("incorrect arguments: %v", args)
	}

	ldconfigPath := args[1]
	if ldconfigPath == "" {
		return nil, args, fmt.Errorf("an ldconfig path must be specified")
	}

	inRoot := args[2]
	if inRoot == "" || inRoot == "/" {
		return nil, args, fmt.Errorf("ldconfig must be run in the non-system root")
	}

	isRootless := false
	if len(args) >= 4 && strings.HasPrefix(args[3], "rootless=") {
		value, err := strconv.ParseBool(strings.TrimPrefix(args[3], "rootless="))
		if err != nil {
			return nil, args, err
		}
		isRootless = value
	}

	remainingArgs := args[3:]
	if isRootless {
		remainingArgs = args[4:]
	}

	l := &Ldconfig{
		ldconfigPath: ldconfigPath,
		inRoot:       inRoot,
		isRootless:   isRootless,
	}
	return l, remainingArgs, nil
}

// CreateSonameSymlinks uses ldconfig to create the soname symlinks in the
// specified directories.
func (l *Ldconfig) CreateSonameSymlinks(directories ...string) error {
	if len(directories) == 0 {
		return nil
	}
	ldconfigPath, err := l.prepareRoot()
	if err != nil {
		return err
	}

	args := []string{
		filepath.Base(ldconfigPath),
		// Explicitly disable updating the LDCache.
		"-N",
		// Specify -n to only process the specified directories.
		"-n",
	}
	args = append(args, directories...)

	return SafeExec(ldconfigPath, args, nil)
}

func (l *Ldconfig) UpdateLDCache(directories ...string) error {
	ldconfigPath, err := l.prepareRoot()
	if err != nil {
		return err
	}

	args := []string{
		filepath.Base(ldconfigPath),
		// Explicitly specify using /etc/ld.so.conf since the host's ldconfig may
		// be configured to use a different config file by default.
		"-f", "/etc/ld.so.conf",
	}

	if l.ldcacheExists() {
		args = append(args, "-C", "/etc/ld.so.cache")
	} else {
		args = append(args, "-N")
	}

	// If the ld.so.conf.d directory exists, we create a config file there
	// containing the required directories, otherwise we add the specified
	// directories to the ldconfig command directly.
	if l.ldsoconfdDirectoryExists() {
		err := createLdsoconfdFile(ldsoconfdFilenamePattern, directories...)
		if err != nil {
			return fmt.Errorf("failed to update ld.so.conf.d: %w", err)
		}
	} else {
		args = append(args, directories...)
	}

	return SafeExec(ldconfigPath, args, nil)
}

func (l *Ldconfig) prepareRoot() (string, error) {
	// To prevent leaking the parent proc filesystem, we create a new proc mount
	// in the specified root.
	if err := mountProc(l.inRoot); err != nil {
		if !l.isRootless {
			return "", fmt.Errorf("error mounting /proc: %w", err)
		}
		log.Printf("Ignoring error for rootless container: %v", err)
	}

	// We mount the host ldconfig before we pivot root since host paths are not
	// visible after the pivot root operation.
	ldconfigPath, err := mountLdConfig(l.ldconfigPath, l.inRoot)
	if err != nil {
		return "", fmt.Errorf("error mounting host ldconfig: %w", err)
	}

	// We pivot to the container root for the new process, this further limits
	// access to the host.
	if err := pivotRoot(l.inRoot); err != nil {
		return "", fmt.Errorf("error running pivot_root: %w", err)
	}

	return ldconfigPath, nil
}

func (l *Ldconfig) ldcacheExists() bool {
	if _, err := os.Stat("/etc/ld.so.cache"); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func (l *Ldconfig) ldsoconfdDirectoryExists() bool {
	info, err := os.Stat("/etc/ld.so.conf.d")
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
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
