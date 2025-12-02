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

package ldconfig

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/moby/sys/mountinfo"
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

// msMoveRoot is used in cases where pivot root is not supported.
// This includes initramfs filesystems where the root is read-only.
// This is adapted from the implementation here:
//
//	https://github.com/opencontainers/runc/blob/e89a29929c775025419ab0d218a43588b4c12b9a/libcontainer/rootfs_linux.go#L1115
//
// With the `mount` and `unmount` calls changed to direct unix.Mount and unix.Unmount calls.
func msMoveRoot(rootfs string) error {
	// Before we move the root and chroot we have to mask all "full" sysfs and
	// procfs mounts which exist on the host. This is because while the kernel
	// has protections against mounting procfs if it has masks, when using
	// chroot(2) the *host* procfs mount is still reachable in the mount
	// namespace and the kernel permits procfs mounts inside --no-pivot
	// containers.
	//
	// Users shouldn't be using --no-pivot except in exceptional circumstances,
	// but to avoid such a trivial security flaw we apply a best-effort
	// protection here. The kernel only allows a mount of a pseudo-filesystem
	// like procfs or sysfs if there is a *full* mount (the root of the
	// filesystem is mounted) without any other locked mount points covering a
	// subtree of the mount.
	//
	// So we try to unmount (or mount tmpfs on top of) any mountpoint which is
	// a full mount of either sysfs or procfs (since those are the most
	// concerning filesystems to us).
	mountinfos, err := mountinfo.GetMounts(func(info *mountinfo.Info) (skip, stop bool) {
		// Collect every sysfs and procfs filesystem, except for those which
		// are non-full mounts or are inside the rootfs of the container.
		if info.Root != "/" ||
			(info.FSType != "proc" && info.FSType != "sysfs") ||
			strings.HasPrefix(info.Mountpoint, rootfs) {
			skip = true
		}
		return
	})
	if err != nil {
		return err
	}
	for _, info := range mountinfos {
		p := info.Mountpoint
		// Be sure umount events are not propagated to the host.
		if err := unix.Mount("", p, "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
			if errors.Is(err, unix.ENOENT) {
				// If the mountpoint doesn't exist that means that we've
				// already blasted away some parent directory of the mountpoint
				// and so we don't care about this error.
				continue
			}
			return err
		}
		if err := unix.Unmount(p, unix.MNT_DETACH); err != nil {
			if !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.EPERM) {
				return err
			} else {
				// If we have not privileges for umounting (e.g. rootless), then
				// cover the path.
				if err := unix.Mount("tmpfs", p, "tmpfs", 0, ""); err != nil {
					return err
				}
			}
		}
	}

	// Move the rootfs on top of "/" in our mount namespace.
	if err := unix.Mount(rootfs, "/", "", unix.MS_MOVE, ""); err != nil {
		return err
	}
	return chroot()
}

func chroot() error {
	if err := unix.Chroot("."); err != nil {
		return &os.PathError{Op: "chroot", Path: ".", Err: err}
	}
	if err := unix.Chdir("/"); err != nil {
		return &os.PathError{Op: "chdir", Path: "/", Err: err}
	}
	return nil
}

// mountLdConfig mounts the host ldconfig to the mount namespace of the hook.
// We use WithProcfd to perform the mount operations to ensure that the changes
// are persisted across the pivot root.
func mountLdConfig(hostLdconfigPath string, containerRoot *os.Root) (string, error) {
	containerRootDirPath := containerRoot.Name()

	hostLdconfigInfo, err := os.Stat(hostLdconfigPath)
	if err != nil {
		return "", fmt.Errorf("error reading host ldconfig: %w", err)
	}

	hookScratchDirPath := "/run/nvidia-ctk-hook/" + uuid.NewString()
	ldconfigPath := filepath.Join(hookScratchDirPath, "ldconfig")
	if err := containerRoot.MkdirAll(hookScratchDirPath[1:], 0755); err != nil {
		return "", fmt.Errorf("error creating hook scratch folder: %w", err)
	}

	//nolint:staticcheck // TODO (ArangoGutierrez): Remove the nolint:staticcheck and properly fix the deprecation warning.
	err = utils.WithProcfd(containerRootDirPath, hookScratchDirPath, func(hookScratchDirFdPath string) error {
		return createTmpFs(hookScratchDirFdPath, int(hostLdconfigInfo.Size()))
	})
	if err != nil {
		return "", fmt.Errorf("error creating tmpfs: %w", err)
	}

	if _, err := containerRoot.OpenFile(ldconfigPath[1:], os.O_CREATE|os.O_RDWR|os.O_TRUNC, hostLdconfigInfo.Mode()); err != nil {
		return "", fmt.Errorf("error creating ldconfig: %w", err)
	}

	//nolint:staticcheck // TODO (ArangoGutierrez): Remove the nolint:staticcheck and properly fix the deprecation warning.
	err = utils.WithProcfd(containerRootDirPath, ldconfigPath, func(ldconfigFdPath string) error {
		return unix.Mount(hostLdconfigPath, ldconfigFdPath, "", unix.MS_BIND|unix.MS_RDONLY|unix.MS_NODEV|unix.MS_PRIVATE|unix.MS_NOSYMFOLLOW, "")
	})
	if err != nil {
		return "", fmt.Errorf("error bind mounting host ldconfig: %w", err)
	}

	return ldconfigPath, nil
}

// mountProc mounts a clean proc filesystem in the new root.
func mountProc(newroot *os.Root) error {
	if err := newroot.MkdirAll("proc", 0755); err != nil {
		return err
	}

	target := filepath.Join(newroot.Name(), "proc")
	return unix.Mount("proc", target, "proc", 0, "")
}

// createTmpFs creates a tmpfs at the specified location with the specified size.
func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}

// createReexecCommand creates a command that can be used to trigger the reexec
// initializer.
// On linux this command runs in new namespaces.
func createReexecCommand(args []string) (*exec.Cmd, error) {
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

	return cmd, nil
}
