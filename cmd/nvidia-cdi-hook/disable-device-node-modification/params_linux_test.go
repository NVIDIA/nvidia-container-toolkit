//go:build linux

/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moby/sys/mountinfo"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestBindMountReadOnly(t *testing.T) {
	requireMountPrivileges(t)

	testCases := map[string]struct {
		mount func(root *os.Root, source *os.File, target *os.File, targetPath string) error
	}{
		"fd-based mount API": {
			mount: func(_ *os.Root, source *os.File, target *os.File, _ string) error {
				return bindMountReadOnly(source, target)
			},
		},
		"legacy mount API": {
			mount: bindMountReadOnlyLegacy,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// A shared parent makes the new mount shared unless it is made private after attaching.
			dir := t.TempDir()
			require.NoError(t, unix.Mount("tmpfs", dir, "tmpfs", 0, ""))
			t.Cleanup(func() {
				_ = unix.Unmount(dir, unix.MNT_DETACH)
			})
			require.NoError(t, unix.Mount("", dir, "", unix.MS_SHARED, ""))

			require.NoError(t, os.WriteFile(filepath.Join(dir, "source"), []byte("modified"), 0400))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "target"), []byte("original"), 0400))

			root, err := os.OpenRoot(dir)
			require.NoError(t, err)
			defer root.Close()

			source, err := root.Open("source")
			require.NoError(t, err)
			defer source.Close()

			target, err := root.Open("target")
			require.NoError(t, err)
			defer target.Close()

			targetPath := filepath.Join(dir, "target")
			t.Cleanup(func() {
				_ = unix.Unmount(targetPath, unix.MNT_DETACH)
			})

			err = tc.mount(root, source, target, "target")
			if errors.Is(err, unix.ENOSYS) {
				t.Skipf("kernel does not support the required mount API: %v", err)
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			require.Equal(t, "modified", string(contents))

			_, err = os.OpenFile(targetPath, os.O_WRONLY, 0)
			require.ErrorIs(t, err, unix.EROFS)

			mounts, err := mountinfo.GetMounts(mountinfo.SingleEntryFilter(targetPath))
			require.NoError(t, err)
			require.Len(t, mounts, 1)
			options := strings.Split(mounts[0].Options, ",")
			require.Contains(t, options, "ro")
			require.Contains(t, options, "nodev")
			require.NotContains(t, mounts[0].Optional, "shared:")
		})
	}
}

// Being root is not enough without CAP_SYS_ADMIN, so probe with a real mount.
func requireMountPrivileges(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := unix.Mount("tmpfs", dir, "tmpfs", 0, ""); err != nil {
		t.Skipf("creating mounts is not permitted: %v", err)
	}
	require.NoError(t, unix.Unmount(dir, unix.MNT_DETACH))
}
