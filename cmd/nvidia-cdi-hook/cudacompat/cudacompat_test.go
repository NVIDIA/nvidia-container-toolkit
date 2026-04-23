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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestCompatLibs(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description                       string
		contents                          map[string]string
		options                           options
		expectedContainerForwardCompatDir string
	}{
		{
			description: "empty root",
			options: options{
				hostDriverVersion: "222.55.66",
			},
		},
		{
			description: "compat lib is newer; no ldcache",
			contents: map[string]string{
				"/usr/local/cuda/compat/libcuda.so.333.88.99": "",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat",
		},
		{
			description: "compat lib is newer; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.333.88.99": "",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat",
		},
		{
			description: "compat lib is older; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.111.88.99": "",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "compat lib has same major version; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "numeric comparison is used; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			options: options{
				hostDriverVersion: "99.55.66",
			},
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat",
		},
		{
			description: "driver version empty; ldcache",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/usr/local/cuda/compat/libcuda.so.222.88.99": "",
			},
			options: options{
				hostDriverVersion: "",
			},
		},
		{
			description: "symlinks are followed",
			contents: map[string]string{
				"/etc/ld.so.cache": "",
				"/etc/alternatives/cuda/compat/libcuda.so.333.88.99": "",
				"/usr/local/cuda": "symlink=/etc/alternatives/cuda",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "/etc/alternatives/cuda/compat",
		},
		{
			description: "symlinks stay in container",
			contents: map[string]string{
				"/etc/ld.so.cache":             "",
				"/compat/libcuda.so.333.88.99": "",
				"/usr/local/cuda":              "symlink=../../../../../../",
			},
			options: options{
				hostDriverVersion: "222.55.66",
			},
			expectedContainerForwardCompatDir: "/compat",
		},
		{
			description: "specified compat path is used",
			contents: map[string]string{
				"/usr/local/cuda/compat/libcuda.so.111.88.99":       "",
				"/usr/local/cuda/compat-other/libcuda.so.333.88.99": "",
			},
			options: options{
				cudaCompatContainerRoot: "/usr/local/cuda/compat-other",
				hostDriverVersion:       "222.55.66",
			},
			expectedContainerForwardCompatDir: "/usr/local/cuda/compat-other",
		},
	}

	for _, tc := range testCases {
		if tc.options.cudaCompatContainerRoot == "" {
			tc.options.cudaCompatContainerRoot = defaultCudaCompatPath
		}

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
			containerForwardCompatDir, err := c.getContainerForwardCompatDir(containerRoot(containerRootDir), &tc.options)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedContainerForwardCompatDir, containerForwardCompatDir)
		})
	}
}

func TestCompatLibsWithElfHeader(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	dataRoot := filepath.Join(moduleRoot, "testdata")

	testCases := []struct {
		description                       string
		options                           options
		expectedContainerForwardCompatDir string
	}{
		{
			description: "container cuda version greater than host cuda version",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostCudaVersion:         "12.8",
			},
			expectedContainerForwardCompatDir: "/compat/575.57.08",
		},
		{
			description: "container cuda version same as host cuda version",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostCudaVersion:         "12.9",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "container cuda version less than host cuda version",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostCudaVersion:         "12.10",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "host driver branch not supported in compat elf header",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostDriverVersion:       "590.44.01",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "host driver branch supported in compat elf header, host driver branch < compat driver branch",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostDriverVersion:       "570.211.01",
			},
			expectedContainerForwardCompatDir: "/compat/575.57.08",
		},
		{
			description: "host driver branch same as compat driver branch, compat driver > host driver",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostDriverVersion:       "575.10.10",
			},
			expectedContainerForwardCompatDir: "/compat/575.57.08",
		},
		{
			description: "host driver branch same as compat driver branch, compat driver = host driver",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostDriverVersion:       "575.57.08",
			},
			expectedContainerForwardCompatDir: "",
		},
		{
			description: "host driver branch same as compat driver branch, compat driver < host driver",
			options: options{
				cudaCompatContainerRoot: "compat/575.57.08",
				hostDriverVersion:       "575.99.99",
			},
			expectedContainerForwardCompatDir: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			containerRootDir := dataRoot
			c := command{
				logger: logger,
			}
			containerForwardCompatDir, err := c.getContainerForwardCompatDir(containerRoot(containerRootDir), &tc.options)
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
