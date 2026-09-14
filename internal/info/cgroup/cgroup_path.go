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

# Portions of this file are derived from github.com/opencontainers/cgroups,
# licensed under the Apache License, Version 2.0.

**/

package cgroup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	v2FsMagicNumber = 0x63677270
	rootDirectory   = "/sys/fs/cgroup"
	unifiedPrefix   = "0::"
)

func IsCgroupV2() bool {
	var s unix.Statfs_t
	_ = unix.Statfs(rootDirectory, &s)
	return s.Type == v2FsMagicNumber
}

func GetAbsolutePath(pid int) (string, error) {
	cgroupProcFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	b, err := os.ReadFile(cgroupProcFile)
	if err != nil {
		return "", err
	}
	return parseCgroupProcFile(b)
}

func parseCgroupProcFile(b []byte) (string, error) {
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, unifiedPrefix) {
			continue
		}
		return filepath.Join(rootDirectory, strings.TrimPrefix(line, unifiedPrefix)), nil
	}
	return "", errors.New("no cgroup v2 unified hierarchy entry found")
}
