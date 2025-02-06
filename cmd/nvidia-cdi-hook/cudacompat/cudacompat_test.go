/*
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
*/

package cudacompat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestCompatLibs(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description                       string
		contents                          map[string]string
		hostDriverVersion                 string
		expectedContainerForwardCompatDir string
	}{
		{
			description:       "empty root",
			hostDriverVersion: "222.55.66",
		},
		{
			description: "compat lib is newer; no ldcache",
			contents: map[string]string{
				"/usr/local/cuda/compat/libcuda.so.333.88.99": "",
			},
			hostDriverVersion: "222.55.66",
		},
		{
			description: "compat lib is newer; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.333.88.99": "",
			},
			hostDriverVersion:                 "222.55.66",
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat",
		},
		{
			description: "compat lib is older; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.111.88.99": "",
			},
			hostDriverVersion:                 "222.55.66",
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "compat lib has same major version; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			hostDriverVersion:                 "222.55.66",
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "numeric comparison is used; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			hostDriverVersion:                 "99.55.66",
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat",
		},
		{
			description: "driver version empty; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			hostDriverVersion: "",
		},
		{
			description: "symlinks are followed",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/etc/alternatives/cuda/compat/libcuda.so.333.88.99": "",
				"/usr/local/cuda": "symlink=/etc/alternatives/cuda",
			},
			hostDriverVersion:                 "222.55.66",
			expectedContainerForwardCompatDir: "/etc/alternatives/cuda/compat",
		},
		{
			description: "symlinks stay in container",
			contents: map[string]string{
				"/etc/ld.so.cache":             "",
				"/compat/libcuda.so.333.88.99": "",
				"/usr/local/cuda":              "symlink=../../../../../../",
			},
			hostDriverVersion:                 "222.55.66",
			expectedContainerForwardCompatDir: "/compat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			containerRootDir := t.TempDir()
			for name, contents := range tc.contents {
				target := filepath.Join(containerRootDir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))

				if strings.HasPrefix(contents, "symlink=") {
					require.NoError(t, os.Symlink(strings.TrimPrefix(contents, "symlink="), target))
					continue
				}

				require.NoError(t, os.WriteFile(target, []byte(contents), 0600))
			}

			c := command{
				logger: logger,
			}
			containerForwardCompatDir, err := c.getContainerForwardCompatDir(containerRoot(containerRootDir), tc.hostDriverVersion)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedContainerForwardCompatDir, containerForwardCompatDir)
		})
	}
}

func TestUpdateLdconfig(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description      string
		folders          []string
		expectedContents string
	}{
		{
			description: "no folders; have no contents",
		},
		{
			description:      "single folder is added",
			folders:          []string{"/usr/local/cuda/compat"},
			expectedContents: "/usr/local/cuda/compat\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			containerRootDir := t.TempDir()
			c := command{
				logger: logger,
			}
			err := c.createLdsoconfdFile(containerRoot(containerRootDir), cudaCompatLdsoconfdFilenamePattern, tc.folders...)
			require.NoError(t, err)

			matches, err := filepath.Glob(filepath.Join(containerRootDir, "/etc/ld.so.conf.d/00-compat-*.conf"))
			require.NoError(t, err)

			if tc.expectedContents == "" {
				require.Empty(t, matches)
				return
			}

			require.Len(t, matches, 1)
			contents, err := os.ReadFile(matches[0])
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedContents, string(contents))
		})
	}

}
