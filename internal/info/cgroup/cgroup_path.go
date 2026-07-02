/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	"fmt"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/cgroups"
	"github.com/opencontainers/cgroups/fs2"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func IsCgroupV2() bool {
	const v2FsMagicNumber = 0x63677270
	var s unix.Statfs_t
	err := unix.Statfs("/sys/fs/cgroup", &s)
	if err != nil {
		panic(err)
	}
	return s.Type == v2FsMagicNumber
}

func GetAbsolutePath(spec specs.Spec) (string, error) {
	c := &cgroups.Cgroup{}
	myCgroupPath := spec.Linux.CgroupsPath
	parts := strings.Split(myCgroupPath, ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("expected cgroupsPath to be of format \"slice:prefix:name\" for systemd cgroups, got %q instead", myCgroupPath)
	}
	c.Parent = parts[0]
	c.ScopePrefix = parts[1]
	c.Name = parts[2]

	return getPath(c)
}

func getUnitName(c *cgroups.Cgroup) string {
	// by default, we create a scope unless the user explicitly asks for a slice.
	if !strings.HasSuffix(c.Name, ".slice") {
		return c.ScopePrefix + "-" + c.Name + ".scope"
	}
	return c.Name
}

func getPath(c *cgroups.Cgroup) (string, error) {

	sliceFull, err := getSliceFull(c)
	if err != nil {
		return "", err
	}

	path := filepath.Join(sliceFull, getUnitName(c))
	path, err = securejoin.SecureJoin(fs2.UnifiedMountpoint, path)
	if err != nil {
		return "", err
	}

	// an example of the final path in rootless:
	// "/sys/fs/cgroup/user.slice/user-1001.slice/user@1001.service/user.slice/libpod-132ff0d72245e6f13a3bbc6cdc5376886897b60ac59eaa8dea1df7ab959cbf1c.scope"

	return path, nil
}

func getSliceFull(c *cgroups.Cgroup) (string, error) {
	slice := "system.slice"
	if c.Rootless {
		slice = "user.slice"
	}
	if c.Parent != "" {
		var err error
		slice, err = expandSlice(c.Parent)
		if err != nil {
			return "", err
		}
	}

	// an example of the final slice in rootless: "/user.slice/user-1001.slice/user@1001.service/user.slice"
	// NOTE: systemdDbus.PropSlice requires the "/user.slice/user-1001.slice/user@1001.service/" prefix NOT to be specified.
	return slice, nil
}

// systemd represents slice hierarchy using `-`, so we need to follow suit when
// generating the path of slice. Essentially, test-a-b.slice becomes
// /test.slice/test-a.slice/test-a-b.slice.
func expandSlice(slice string) (string, error) {
	suffix := ".slice"
	// Name has to end with ".slice", but can't be just ".slice".
	if len(slice) < len(suffix) || !strings.HasSuffix(slice, suffix) {
		return "", fmt.Errorf("invalid slice name: %s", slice)
	}

	// Path-separators are not allowed.
	if strings.Contains(slice, "/") {
		return "", fmt.Errorf("invalid slice name: %s", slice)
	}

	var path, prefix string
	sliceName := strings.TrimSuffix(slice, suffix)
	// if input was -.slice, we should just return root now
	if sliceName == "-" {
		return "/", nil
	}
	for component := range strings.SplitSeq(sliceName, "-") {
		// test--a.slice isn't permitted, nor is -test.slice.
		if component == "" {
			return "", fmt.Errorf("invalid slice name: %s", slice)
		}

		// Append the component to the path and to the prefix.
		path += "/" + prefix + component + suffix
		prefix += component + "-"
	}
	return path, nil
}
