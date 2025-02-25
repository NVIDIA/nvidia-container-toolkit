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

package ldcache

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/dmz"
)

// SafeExec attempts to clone the specified binary (as an memfd, for example) before executing it.
func (m command) SafeExec(path string, args []string, envv []string) error {
	safeExe, err := cloneBinary(path)
	if err != nil {
		m.logger.Warningf("Failed to clone binary %q: %v; falling back to Exec", path, err)
		//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
		return syscall.Exec(path, args, envv)
	}
	defer safeExe.Close()

	exePath := "/proc/self/fd/" + strconv.Itoa(int(safeExe.Fd()))
	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	return syscall.Exec(exePath, args, envv)
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

	return dmz.CloneBinary(exe, size, path, os.TempDir())
}
