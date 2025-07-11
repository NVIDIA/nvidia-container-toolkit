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

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
)

func createParamsFileInContainer(containerRootDirPath string, contents []byte) error {
	hookScratchDirPath := "/var/run/nvidia-ctk-hook"
	if err := utils.MkdirAllInRoot(containerRootDirPath, hookScratchDirPath, 0755); err != nil {
		return fmt.Errorf("error creating hook scratch folder: %w", err)
	}

	err := utils.WithProcfd(containerRootDirPath, hookScratchDirPath, func(hookScratchDirFdPath string) error {
		return createTmpFs(hookScratchDirFdPath, len(contents))

	})
	if err != nil {
		return fmt.Errorf("failed to create tmpfs mount for params file: %w", err)
	}

	modifiedParamsFilePath := filepath.Join(hookScratchDirPath, "nvct-params")
	if _, err := createFileInRoot(containerRootDirPath, modifiedParamsFilePath, 0444); err != nil {
		return fmt.Errorf("error creating modified params file: %w", err)
	}

	err = utils.WithProcfd(containerRootDirPath, modifiedParamsFilePath, func(modifiedParamsFileFdPath string) error {
		modifiedParamsFile, err := os.OpenFile(modifiedParamsFileFdPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0444)
		if err != nil {
			return fmt.Errorf("failed to open modified params file: %w", err)
		}
		defer modifiedParamsFile.Close()

		if _, err := modifiedParamsFile.Write(contents); err != nil {
			return fmt.Errorf("failed to write temporary params file: %w", err)
		}

		err = utils.WithProcfd(containerRootDirPath, nvidiaDriverParamsPath, func(nvidiaDriverParamsFdPath string) error {
			return unix.Mount(modifiedParamsFileFdPath, nvidiaDriverParamsFdPath, "", unix.MS_BIND|unix.MS_RDONLY|unix.MS_NODEV|unix.MS_PRIVATE|unix.MS_NOSYMFOLLOW, "")
		})
		if err != nil {
			return fmt.Errorf("failed to mount modified params file: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}

// TODO(ArangoGutierrez): This function also exists in internal/ldconfig we should move this to a separate package.
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
