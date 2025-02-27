//go:build linux

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

package create_soname_symlinks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	securejoin "github.com/cyphar/filepath-securejoin"

	"github.com/moby/sys/reexec"
	"github.com/opencontainers/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
)

// pivotRoot will call pivot_root such that rootfs becomes the new root
// filesystem, and everything else is cleaned up.
// This is adapted from the implementation here:
//
//	https://github.com/opencontainers/runc/blob/e89a29929c775025419ab0d218a43588b4c12b9a/libcontainer/rootfs_linux.go#L1056-L1113
//
// With the `mount` and `unmount` calls changed to direct unix.Mount and unix.Unmount calls.
func pivotRoot(rootfs string) error {
	// While the documentation may claim otherwise, pivot_root(".", ".") is
	// actually valid. What this results in is / being the new root but
	// /proc/self/cwd being the old root. Since we can play around with the cwd
	// with pivot_root this allows us to pivot without creating directories in
	// the rootfs. Shout-outs to the LXC developers for giving us this idea.

	oldroot, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: "/", Err: err}
	}
	defer unix.Close(oldroot) //nolint: errcheck

	newroot, err := unix.Open(rootfs, unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: rootfs, Err: err}
	}
	defer unix.Close(newroot) //nolint: errcheck

	// Change to the new root so that the pivot_root actually acts on it.
	if err := unix.Fchdir(newroot); err != nil {
		return &os.PathError{Op: "fchdir", Path: "fd " + strconv.Itoa(newroot), Err: err}
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return &os.PathError{Op: "pivot_root", Path: ".", Err: err}
	}

	// Currently our "." is oldroot (according to the current kernel code).
	// However, purely for safety, we will fchdir(oldroot) since there isn't
	// really any guarantee from the kernel what /proc/self/cwd will be after a
	// pivot_root(2).

	if err := unix.Fchdir(oldroot); err != nil {
		return &os.PathError{Op: "fchdir", Path: "fd " + strconv.Itoa(oldroot), Err: err}
	}

	// Make oldroot rslave to make sure our unmounts don't propagate to the
	// host (and thus bork the machine). We don't use rprivate because this is
	// known to cause issues due to races where we still have a reference to a
	// mount while a process in the host namespace are trying to operate on
	// something they think has no mounts (devicemapper in particular).
	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return err
	}
	// Perform the unmount. MNT_DETACH allows us to unmount /proc/self/cwd.
	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return err
	}

	// Switch back to our shiny new root.
	if err := unix.Chdir("/"); err != nil {
		return &os.PathError{Op: "chdir", Path: "/", Err: err}
	}
	return nil
}

// mountLdConfig mounts the host ldconfig to the mount namespace of the hook.
// We use WithProcfd to perform the mount operations to ensure that the changes
// are persisted across the pivot root.
func mountLdConfig(hostLdconfigPath string, containerRootDirPath string) (string, error) {
	hostLdconfigInfo, err := os.Stat(hostLdconfigPath)
	if err != nil {
		return "", fmt.Errorf("error reading host ldconfig: %w", err)
	}

	hookScratchDirPath := "/var/run/nvidia-ctk-hook"
	ldconfigPath := filepath.Join(hookScratchDirPath, "ldconfig")
	if err := utils.MkdirAllInRoot(containerRootDirPath, hookScratchDirPath, 0755); err != nil {
		return "", fmt.Errorf("error creating hook scratch folder: %w", err)
	}

	err = utils.WithProcfd(containerRootDirPath, hookScratchDirPath, func(hookScratchDirFdPath string) error {
		return createTmpFs(hookScratchDirFdPath, int(hostLdconfigInfo.Size()))

	})
	if err != nil {
		return "", fmt.Errorf("error creating tmpfs: %w", err)
	}

	if _, err := createFileInRoot(containerRootDirPath, ldconfigPath, hostLdconfigInfo.Mode()); err != nil {
		return "", fmt.Errorf("error creating ldconfig: %w", err)
	}

	err = utils.WithProcfd(containerRootDirPath, ldconfigPath, func(ldconfigFdPath string) error {
		return unix.Mount(hostLdconfigPath, ldconfigFdPath, "", unix.MS_BIND|unix.MS_RDONLY|unix.MS_NODEV|unix.MS_PRIVATE|unix.MS_NOSYMFOLLOW, "")
	})
	if err != nil {
		return "", fmt.Errorf("error bind mounting host ldconfig: %w", err)
	}

	return ldconfigPath, nil
}

func createFileInRoot(containerRootDirPath string, destinationPath string, mode os.FileMode) (string, error) {
	dest, err := securejoin.SecureJoin(containerRootDirPath, destinationPath)
	if err != nil {
		return "", err
	}
	// Make the parent directory.
	destDir, destBase := filepath.Split(dest)
	destDirFd, err := utils.MkdirAllInRootOpen(containerRootDirPath, destDir, 0755)
	if err != nil {
		return "", fmt.Errorf("error creating parent dir: %w", err)
	}
	defer destDirFd.Close()
	// Make the target file. We want to avoid opening any file that is
	// already there because it could be a "bad" file like an invalid
	// device or hung tty that might cause a DoS, so we use mknodat.
	// destBase does not contain any "/" components, and mknodat does
	// not follow trailing symlinks, so we can safely just call mknodat
	// here.
	if err := unix.Mknodat(int(destDirFd.Fd()), destBase, unix.S_IFREG|uint32(mode), 0); err != nil {
		// If we get EEXIST, there was already an inode there and
		// we can consider that a success.
		if !errors.Is(err, unix.EEXIST) {
			return "", fmt.Errorf("error creating empty file: %w", err)
		}
	}
	return dest, nil
}

// mountProc mounts a clean proc filesystem in the new root.
func mountProc(newroot string) error {
	target := filepath.Join(newroot, "/proc")

	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	return unix.Mount("proc", target, "proc", 0, "")
}

// createTmpFs creates a tmpfs at the specified location with the specified size.
func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}

// createReexecCommand creates a command that can be used to trigger the reexec
// initializer.
// On linux this command runs in new namespaces.
func createReexecCommand(args []string) *exec.Cmd {
	cmd := reexec.Command(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNET,
	}

	return cmd
}
