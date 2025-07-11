//go:build linux

/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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
	"os"
	"strconv"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/exeseal"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// SafeExec attempts to clone the specified binary (as an memfd, for example) before executing it.
func SafeExec(path string, args []string, envv []string) error {
	safeExe, err := cloneBinary(path)
	if err != nil {
		return syscall.Exec(path, oci.Escape(args), envv) //nolint:gosec
	}
	defer safeExe.Close()

	exePath := "/proc/self/fd/" + strconv.Itoa(int(safeExe.Fd()))
	return syscall.Exec(exePath, oci.Escape(args), envv) //nolint:gosec
}

func cloneBinary(path string) (*os.File, error) {
	exe, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening current binary: %w", err)
	}
	defer exe.Close()

	stat, err := exe.Stat()
	if err != nil {
		return nil, fmt.Errorf("checking %v size: %w", path, err)
	}
	size := stat.Size()

	return exeseal.CloneBinary(exe, size, path, os.TempDir())
}
