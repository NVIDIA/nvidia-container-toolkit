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

	if err := unix.Mount(utils.GetProcFdPath(modifiedParamsFile), utils.GetProcFdPath(nvidiaDriverParamsFile), "",
		unix.MS_BIND|unix.MS_RDONLY|unix.MS_NODEV|unix.MS_PRIVATE|unix.MS_NOSYMFOLLOW, ""); err != nil {
		return fmt.Errorf("failed to mount modified params file: %w", err)
	}

	return nil
}

func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}
