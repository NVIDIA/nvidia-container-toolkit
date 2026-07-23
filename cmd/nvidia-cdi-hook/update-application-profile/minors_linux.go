/**
# SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package updateapplicationprofile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// readDirBatchSize is the number of directory entries read from the
// container's /dev directory per ReadDir call.
const readDirBatchSize = 100

// listDeviceNodes enumerates the entries under the container's /dev directory.
// The char-device check uses the cheap readdir type; the minor number is
// resolved for char-device entries.
func listDeviceNodes(containerRoot *os.Root) ([]deviceNode, error) {
	devDir, err := containerRoot.Open("dev")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open /dev in container: %w", err)
	}
	defer devDir.Close()

	var nodes []deviceNode
	for {
		entries, readErr := devDir.ReadDir(readDirBatchSize)
		for _, entry := range entries {
			node := deviceNode{
				name:         entry.Name(),
				isCharDevice: entry.Type()&os.ModeCharDevice != 0,
			}
			if node.isCharDevice {
				minor, err := deviceNodeMinor(containerRoot, entry.Name())
				if err != nil {
					return nil, err
				}
				node.minor = minor
			}
			nodes = append(nodes, node)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to read /dev in container: %w", readErr)
		}
	}

	return nodes, nil
}

// deviceNodeMinor returns the device minor number for the named entry under the
// container's /dev directory.
func deviceNodeMinor(containerRoot *os.Root, name string) (int, error) {
	info, err := containerRoot.Lstat(filepath.Join("dev", name))
	if err != nil {
		return 0, fmt.Errorf("failed to stat %s: %w", name, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("unexpected stat type for %s", name)
	}
	return int(unix.Minor(stat.Rdev)), nil
}
