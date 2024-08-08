/**
# Copyright 2024 NVIDIA CORPORATION
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

package containerd

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	runningSystemd bool
	detectSystemd  sync.Once
)

// hasSystemd returns whether systemd is running on this system.
// This is adapted from the containerd implementation.
// https://github.com/containerd/containerd/blob/1fb1882c7de7239711c343c65fa266a2f1eba9db/internal/cri/server/runtime_config_linux.go#L63-L68
func (b *builder) hasSystemd() bool {
	detectSystemd.Do(func() {
		fi, err := os.Lstat(filepath.Join(b.hostRoot, "/run/systemd/system"))
		if err != nil && !os.IsNotExist(err) {
			b.logger.Warningf("unexpected error detecting systemd: %v; assuming no systemd", err)
			return
		}
		runningSystemd = err == nil && fi.IsDir()
	})
	return runningSystemd
}
