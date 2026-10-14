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

package discover

import (
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

func TestIPCMounts(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	l := ipcMounts(
		mounts{
			logger: logger,
			lookup: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					return []string{"/host/path"}, nil
				},
			},
			required: []string{"target"},
		},
	)

	mounts, err := l.Mounts()
	require.NoError(t, err)

	require.EqualValues(
		t,
		[]Mount{
			{
				HostPath: "/host/path",
				Path:     "/host/path",
				Options: []string{
					"nosuid",
					"nodev",
					"rbind",
					"rprivate",
					"noexec",
				},
			},
		},
		mounts,
	)
}

func TestNewIPCDiscoverer(t *testing.T) {
	ipcOptions := []string{"nosuid", "nodev", "rbind", "rprivate", "noexec"}

	testCases := []struct {
		description    string
		files          []string
		dirs           []string
		expectedMounts []Mount
	}{
		{
			description: "no IPC paths present",
		},
		{
			description: "MPS pipe directory is discovered",
			dirs:        []string{"/tmp/nvidia-mps"},
			expectedMounts: []Mount{
				{Path: "/tmp/nvidia-mps", Options: ipcOptions},
			},
		},
		{
			description: "MPS pipe path that is a regular file is ignored",
			files:       []string{"/tmp/nvidia-mps"},
		},
		{
			description: "sockets are discovered",
			files: []string{
				"/run/nvidia-persistenced/socket",
				"/var/run/nvidia-fabricmanager/socket",
			},
			expectedMounts: []Mount{
				{Path: "/run/nvidia-persistenced/socket", Options: ipcOptions},
				{Path: "/var/run/nvidia-fabricmanager/socket", Options: ipcOptions},
			},
		},
		{
			description: "socket paths that are directories are ignored",
			dirs: []string{
				"/run/nvidia-persistenced/socket",
				"/run/nvidia-fabricmanager/socket",
			},
		},
		{
			description: "sockets and MPS pipe directory are discovered",
			files: []string{
				"/run/nvidia-persistenced/socket",
				"/run/nvidia-fabricmanager/socket",
			},
			dirs: []string{"/tmp/nvidia-mps"},
			expectedMounts: []Mount{
				{Path: "/run/nvidia-persistenced/socket", Options: ipcOptions},
				{Path: "/run/nvidia-fabricmanager/socket", Options: ipcOptions},
				{Path: "/tmp/nvidia-mps", Options: ipcOptions},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := testlog.NewNullLogger()
			driverRoot := t.TempDir()

			for _, d := range tc.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(driverRoot, d), 0755))
			}
			for _, f := range tc.files {
				path := filepath.Join(driverRoot, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, nil, 0600))
			}

			d, err := NewIPCDiscoverer(logger, driverRoot)
			require.NoError(t, err)

			var expectedMounts []Mount
			for _, m := range tc.expectedMounts {
				m.HostPath = filepath.Join(driverRoot, m.Path)
				expectedMounts = append(expectedMounts, m)
			}

			mounts, err := d.Mounts()
			require.NoError(t, err)
			require.EqualValues(t, expectedMounts, mounts)
		})
	}
}
