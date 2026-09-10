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
)

func IsCgroupV2() bool {
	var s unix.Statfs_t
	err := unix.Statfs(rootDirectory, &s)
	if err != nil {
		panic(err)
	}
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
	parts := strings.Split(strings.TrimSpace(string(b)), ":")
	if len(parts) != 3 {
		return "", errors.New("invalid cgroup path")
	}
	return filepath.Join(rootDirectory, parts[2]), nil
}
