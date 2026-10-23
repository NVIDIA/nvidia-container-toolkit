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

package disabledevicenodemodification

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/utils"
)

func createParamsFileInContainer(containerRoot *os.Root, contents []byte) error {

	hookScratchDirPath := filepath.Join("/run/nvidia-ctk-hook", uuid.NewString())
	if err := containerRoot.MkdirAll(hookScratchDirPath[1:], 0755); err != nil {
		return fmt.Errorf("error creating hook scratch folder: %w", err)
	}

	hookScratchDir, err := containerRoot.Open(hookScratchDirPath[1:])
	if err != nil {
		return fmt.Errorf("error opening hook scratch folder: %w", err)
	}
	defer hookScratchDir.Close()
	if err := createTmpFs(utils.GetProcFdPath(hookScratchDir), len(contents)); err != nil {
		return fmt.Errorf("failed to create tmpfs mount for params file: %w", err)
	}

	modifiedParamsFilePath := filepath.Join(hookScratchDirPath, "nvct-params")
	modifiedParamsFile, err := containerRoot.OpenFile(modifiedParamsFilePath[1:], os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0444)
	if err != nil {
		return fmt.Errorf("error creating modified params file: %w", err)
	}
	defer modifiedParamsFile.Close()

	if _, err := modifiedParamsFile.Write(contents); err != nil {
		return fmt.Errorf("failed to write temporary params file: %w", err)
	}

	nvidiaDriverParamsFile, err := containerRoot.OpenFile(nvidiaDriverParamsPath[1:], os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error opening nvidia driver params file: %w", err)
	}
	defer nvidiaDriverParamsFile.Close()

	err = bindMountReadOnly(modifiedParamsFile, nvidiaDriverParamsFile)
	if errors.Is(err, unix.ENOSYS) {
		return bindMountReadOnlyLegacy(containerRoot, modifiedParamsFile, nvidiaDriverParamsFile, nvidiaDriverParamsPath[1:])
	}
	return err
}

func bindMountReadOnly(source *os.File, target *os.File) error {
	treeFd, err := unix.OpenTree(int(source.Fd()), "",
		unix.AT_EMPTY_PATH|unix.OPEN_TREE_CLONE|unix.OPEN_TREE_CLOEXEC)
	if err != nil {
		return fmt.Errorf("failed to clone modified params file mount: %w", err)
	}
	defer unix.Close(treeFd)

	if err := unix.MountSetattr(treeFd, "",
		unix.AT_EMPTY_PATH,
		&unix.MountAttr{
			Attr_set: unix.MOUNT_ATTR_RDONLY | unix.MOUNT_ATTR_NODEV,
		},
	); err != nil {
		return fmt.Errorf("failed to set attributes on modified params file mount: %w", err)
	}

	if err := unix.MoveMount(treeFd, "", int(target.Fd()), "",
		unix.MOVE_MOUNT_F_EMPTY_PATH|unix.MOVE_MOUNT_T_EMPTY_PATH); err != nil {
		return fmt.Errorf("failed to attach modified params file mount: %w", err)
	}

	// A shared parent overrides propagation set before attaching, so we have to set it to private afterwards.
	if err := unix.MountSetattr(treeFd, "",
		unix.AT_EMPTY_PATH,
		&unix.MountAttr{
			Propagation: unix.MS_PRIVATE,
		},
	); err != nil {
		return fmt.Errorf("failed to make modified params file mount private: %w", err)
	}

	return nil
}

func bindMountReadOnlyLegacy(containerRoot *os.Root, source *os.File, target *os.File, targetPath string) error {
	// mount(2) ignores MS_RDONLY, MS_NODEV and MS_NOSYMFOLLOW alongside MS_BIND.
	if err := unix.Mount(utils.GetProcFdPath(source), utils.GetProcFdPath(target), "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to mount modified params file: %w", err)
	}

	// target still refers to the file beneath the new mount.
	mounted, err := containerRoot.OpenFile(targetPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error reopening mounted params file: %w", err)
	}
	defer mounted.Close()

	if err := unix.Mount("", utils.GetProcFdPath(mounted), "",
		unix.MS_REMOUNT|unix.MS_BIND|unix.MS_RDONLY|unix.MS_NODEV|unix.MS_NOSYMFOLLOW, ""); err != nil {
		return fmt.Errorf("failed to remount modified params file as read-only: %w", err)
	}
	if err := unix.Mount("", utils.GetProcFdPath(mounted), "", unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make modified params file mount private: %w", err)
	}
	return nil
}

func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}
