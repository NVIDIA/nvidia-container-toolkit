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
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/opencontainers/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
)

func createParamsFileInContainer(containerRoot *os.Root, contents []byte) error {
	containerRootDirPath := containerRoot.Name()

	hookScratchDirPath := "/run/nvidia-ctk-hook" + uuid.NewString()
	if err := containerRoot.MkdirAll(hookScratchDirPath[1:], 0755); err != nil {
		return fmt.Errorf("error creating hook scratch folder: %w", err)
	}

	//nolint:staticcheck // TODO (ArangoGutierrez): Remove the nolint:staticcheck and properly fix the deprecation warning.
	err := utils.WithProcfd(containerRootDirPath, hookScratchDirPath, func(hookScratchDirFdPath string) error {
		return createTmpFs(hookScratchDirFdPath, len(contents))
	})
	if err != nil {
		return fmt.Errorf("failed to create tmpfs mount for params file: %w", err)
	}

	modifiedParamsFilePath := filepath.Join(hookScratchDirPath, "nvct-params")
	if _, err := containerRoot.OpenFile(modifiedParamsFilePath[1:], os.O_CREATE|os.O_RDONLY|os.O_TRUNC, 0444); err != nil {
		return fmt.Errorf("error creating modified params file: %w", err)
	}

	//nolint:staticcheck // TODO (ArangoGutierrez): Remove the nolint:staticcheck and properly fix the deprecation warning.
	err = utils.WithProcfd(containerRootDirPath, modifiedParamsFilePath, func(modifiedParamsFileFdPath string) error {
		modifiedParamsFile, err := os.OpenFile(modifiedParamsFileFdPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0444)
		if err != nil {
			return fmt.Errorf("failed to open modified params file: %w", err)
		}
		defer modifiedParamsFile.Close()

		if _, err := modifiedParamsFile.Write(contents); err != nil {
			return fmt.Errorf("failed to write temporary params file: %w", err)
		}

		//nolint:staticcheck // TODO (ArangoGutierrez): Remove the nolint:staticcheck and properly fix the deprecation warning.
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
