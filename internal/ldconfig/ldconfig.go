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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs"

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
	noPivotRoot  bool
	directories  []string
}

// NewRunner creates an exec.Cmd that can be used to run ldconfig.
func NewRunner(id string, ldconfigPath string, containerRoot string, additionalargs ...string) (*exec.Cmd, error) {
	args := []string{
		id,
		"--ldconfig-path", strings.TrimPrefix(config.NormalizeLDConfigPath("@"+ldconfigPath), "@"),
		"--container-root", containerRoot,
	}

	if noPivotRoot() {
		args = append(args, "--no-pivot")
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
// The following optional flags are supported:
//
//	--no-pivot	pivot_root should not be used to provide process isolation.
//
// The remaining args are folders where soname symlinks need to be created.
func NewFromArgs(args ...string) (*Ldconfig, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("incorrect arguments: %v", args)
	}
	fs := flag.NewFlagSet(args[1], flag.ExitOnError)
	ldconfigPath := fs.String("ldconfig-path", "", "the path to ldconfig on the host")
	containerRoot := fs.String("container-root", "", "the path in which ldconfig must be run")
	noPivot := fs.Bool("no-pivot", false, "don't use pivot_root to perform isolation")
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
		ldconfigPath: *ldconfigPath,
		inRoot:       *containerRoot,
		noPivotRoot:  *noPivot,
		directories:  fs.Args(),
	}
	return l, nil
}

func (l *Ldconfig) UpdateLDCache() error {
	ldconfigPath, err := l.prepareRoot()
	if err != nil {
		return err
	}

	args := []string{
		filepath.Base(ldconfigPath),
		// Explicitly specify using /etc/ld.so.conf since the host's ldconfig may
		// be configured to use a different config file by default.
		"-f", "/etc/ld.so.conf",
		"-C", "/etc/ld.so.cache",
	}

	if err := createLdsoconfdFile(ldsoconfdFilenamePattern, l.directories...); err != nil {
		return fmt.Errorf("failed to update ld.so.conf.d: %w", err)
	}

	return SafeExec(ldconfigPath, args, nil)
}

func (l *Ldconfig) prepareRoot() (string, error) {
	// To prevent leaking the parent proc filesystem, we create a new proc mount
	// in the specified root.
	if err := mountProc(l.inRoot); err != nil {
		return "", fmt.Errorf("error mounting /proc: %w", err)
	}

	// We mount the host ldconfig before we pivot root since host paths are not
	// visible after the pivot root operation.
	ldconfigPath, err := mountLdConfig(l.ldconfigPath, l.inRoot)
	if err != nil {
		return "", fmt.Errorf("error mounting host ldconfig: %w", err)
	}

	// We select the function to pivot the root based on whether pivot_root is
	// supported.
	// See https://github.com/opencontainers/runc/blob/c3d127f6e8d9f6c06d78b8329cafa8dd39f6236e/libcontainer/rootfs_linux.go#L207-L216
	var pivotRootFn func(string) error
	if l.noPivotRoot {
		pivotRootFn = msMoveRoot
	} else {
		pivotRootFn = pivotRoot
	}
	// We pivot to the container root for the new process, this further limits
	// access to the host.
	if err := pivotRootFn(l.inRoot); err != nil {
		return "", fmt.Errorf("error running pivot_root: %w", err)
	}

	return ldconfigPath, nil
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

// noPivotRoot checks whether the current root filesystem supports a pivot_root.
// See https://github.com/opencontainers/runc/blob/main/libcontainer/SPEC.md#filesystem
// for a discussion on when this is not the case.
// If we fail to detect whether pivot-root is supported, we assume that it is supported.
// The logic to check for support is adapted from kata-containers:
//
//	https://github.com/kata-containers/kata-containers/blob/e7b9eddcede4bbe2edeb9c3af7b2358dc65da76f/src/agent/src/sandbox.rs#L150
//
// and checks whether "/" is mounted as a rootfs.
func noPivotRoot() bool {
	rootFsType, err := getRootfsType("/")
	if err != nil {
		return false
	}
	return rootFsType == "rootfs"
}

func getRootfsType(path string) (string, error) {
	procSelf, err := procfs.Self()
	if err != nil {
		return "", err
	}

	mountStats, err := procSelf.MountStats()
	if err != nil {
		return "", err
	}

	for _, mountStat := range mountStats {
		if mountStat.Mount == path {
			return mountStat.Type, nil
		}
	}
	return "", fmt.Errorf("mount stats for %q not found", path)
}
