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
**/

package ldconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

func TestCreateMuslPathFileIfRequired(t *testing.T) {
	arch, ok := muslArchs[runtime.GOARCH]
	if !ok {
		t.Skipf("no musl dynamic linker name for %v", runtime.GOARCH)
	}
	loader := "/lib/ld-musl-" + arch + ".so.1"
	pathFile := "/etc/ld-musl-" + arch + ".path"

	testCases := []struct {
		description string
		files       map[string]string
		driverDirs  []string
		systemDirs  []string
		expected    *string
	}{
		{
			description: "glibc containers get no .path file",
			files:       map[string]string{"/etc/debian_version": ""},
			driverDirs:  []string{"/usr/lib/x86_64-linux-gnu"},
		},
		{
			description: "a new file keeps musl's default search path after the driver directories",
			files:       map[string]string{"/etc/alpine-release": ""},
			driverDirs:  []string{"/usr/lib/x86_64-linux-gnu", "/usr/lib/x86_64-linux-gnu/vdpau"},
			systemDirs:  []string{"/lib64", "/usr/lib64"},
			expected:    new("/usr/lib/x86_64-linux-gnu\n/usr/lib/x86_64-linux-gnu/vdpau\n/lib\n/usr/local/lib\n/usr/lib\n/lib64\n/usr/lib64\n"),
		},
		{
			description: "musl is detected by its dynamic linker",
			files:       map[string]string{loader: ""},
			driverDirs:  []string{"/usr/lib/x86_64-linux-gnu"},
			expected:    new("/usr/lib/x86_64-linux-gnu\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description: "existing entries, colon or newline separated, keep their order after the driver directories",
			files: map[string]string{
				"/etc/alpine-release": "",
				pathFile:              "/opt/a:/opt/b\n/usr/lib\n",
			},
			driverDirs: []string{"/usr/lib/x86_64-linux-gnu", "/usr/lib"},
			systemDirs: []string{"/usr/lib", "/lib64"},
			expected:   new("/usr/lib/x86_64-linux-gnu\n/opt/a\n/opt/b\n/usr/lib\n/lib64\n"),
		},
		{
			description: "the compat32 directory is never added",
			files:       map[string]string{"/etc/alpine-release": ""},
			driverDirs:  []string{"/usr/lib/x86_64-linux-gnu", discover.Compat32Directory},
			expected:    new("/usr/lib/x86_64-linux-gnu\n/lib\n/usr/local/lib\n/usr/lib\n"),
		},
		{
			description: "a second update adds nothing twice",
			files: map[string]string{
				"/etc/alpine-release": "",
				pathFile:              "/usr/lib/x86_64-linux-gnu\n/lib\n/usr/local/lib\n/usr/lib\n/lib64\n",
			},
			driverDirs: []string{"/usr/lib/x86_64-linux-gnu"},
			systemDirs: []string{"/lib64"},
			expected:   new("/usr/lib/x86_64-linux-gnu\n/lib\n/usr/local/lib\n/usr/lib\n/lib64\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "etc"), 0o755))
			for name, contents := range tc.files {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600))
			}

			require.NoError(t, createMuslPathFileIfRequired(root, tc.driverDirs, tc.systemDirs))

			contents, err := os.ReadFile(filepath.Join(root, pathFile))
			if tc.expected == nil {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			require.Equal(t, *tc.expected, string(contents))
		})
	}
}
