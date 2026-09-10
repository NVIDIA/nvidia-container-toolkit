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
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/moby/sys/mountinfo"
	"github.com/moby/sys/reexec"
	"github.com/opencontainers/runc/libcontainer/exeseal"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/utils"
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

// maskPseudoFilesystems mounts an empty tmpfs over root/proc and root/sys,
// if present, before pivoting into root. ldconfig does not need either
// filesystem for its own operation, so rather than relying on that (an
// assumption about the behavior of an unaudited third-party binary, today
// and in every future version of it), this makes both unconditionally
// inert: whatever either path might otherwise expose post-pivot (a stale
// mount carried in from outside, or static content baked into the
// container image itself) can never be read by anything running in the
// isolated namespaces below, because there is nothing real there to read.
func maskPseudoFilesystems(root *os.Root) error {
	for _, name := range []string{"proc", "sys"} {
		f, err := root.Open(name)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("error opening %s: %w", name, err)
		}
		defer f.Close()

		if err := unix.Mount("tmpfs", utils.GetProcFdPath(f), "tmpfs", 0, ""); err != nil {
			return fmt.Errorf("error masking %s: %w", name, err)
		}
	}
	return nil
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

// cloneLdconfigIntoMemfd copies the contents of hostLdconfigPath into a
// sealed, anonymous memfd (falling back to an unlinked tmpfile only if
// memfd_create is unavailable), via libcontainer/exeseal.CloneBinary. Done
// before any namespace changes or pivot -- nothing attacker-controlled is
// reachable yet, so there is no TOCTOU race on hostLdconfigPath itself.
//
// The returned file has no underlying path or mount-namespace membership:
// it survives pivot_root untouched and can be exec'd directly by
// descriptor (see SafeExec).
func cloneLdconfigIntoMemfd(hostLdconfigPath string) (*os.File, error) {
	src, err := os.Open(hostLdconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error opening host ldconfig: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return nil, fmt.Errorf("error statting host ldconfig: %w", err)
	}

	return exeseal.CloneBinary(src, info.Size(), "ldconfig", os.TempDir())
}

// SafeExec replaces the calling process with the program held in file (a
// sealed memfd, see cloneLdconfigIntoMemfd), executed by descriptor via
// execveat(fd, "", argv, envp, AT_EMPTY_PATH) -- no path lookup, so this
// works identically whether or not /proc is mounted anywhere in the
// caller's current namespaces. Never returns on success.
func SafeExec(file *os.File, argv []string, envv []string) error {
	argvp, err := syscall.SlicePtrFromStrings(argv)
	if err != nil {
		return fmt.Errorf("error converting argv: %w", err)
	}
	envvp, err := syscall.SlicePtrFromStrings(envv)
	if err != nil {
		return fmt.Errorf("error converting envp: %w", err)
	}
	emptyPath, err := unix.BytePtrFromString("")
	if err != nil {
		return err
	}

	_, _, errno := unix.Syscall6(
		unix.SYS_EXECVEAT,
		file.Fd(),
		uintptr(unsafe.Pointer(emptyPath)),
		uintptr(unsafe.Pointer(&argvp[0])),
		uintptr(unsafe.Pointer(&envvp[0])),
		uintptr(unix.AT_EMPTY_PATH),
		0,
	)
	// file.Fd() only returns the raw descriptor number; keep file itself
	// reachable until the syscall above completes, otherwise the GC could
	// finalize (and close) it mid-syscall.
	runtime.KeepAlive(file)
	if errno != 0 {
		return fmt.Errorf("execveat: %w", errno)
	}
	return nil
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
