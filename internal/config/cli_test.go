/**
# Copyright 2023 NVIDIA CORPORATION
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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeLDConfigPath(t *testing.T) {
	testDir := t.TempDir()

	f, err := os.Create(filepath.Join(testDir, "exists.real"))
	require.NoError(t, err)
	_ = f.Close()

	testCases := []struct {
		description string
		ldconfig    string
		expected    string
	}{
		{
			description: "empty input",
		},
		{
			description: "non-host with .real suffix returns as is",
			ldconfig:    "/some/path/ldconfig.real",
			expected:    "/some/path/ldconfig.real",
		},
		{
			description: "non-host without .real suffix returns as is",
			ldconfig:    "/some/path/ldconfig",
			expected:    "/some/path/ldconfig",
		},
		{
			description: "host .real file exists is returned",
			ldconfig:    "@" + filepath.Join(testDir, "exists.real"),
			expected:    "@" + filepath.Join(testDir, "exists.real"),
		},
		{
			description: "host resolves .real file",
			ldconfig:    "@" + filepath.Join(testDir, "exists"),
			expected:    "@" + filepath.Join(testDir, "exists.real"),
		},
		{
			description: "host .real file not exists strips suffix",
			ldconfig:    "@/does/not/exist.real",
			expected:    "@/does/not/exist",
		},
		{
			description: "host file returned as is if no .real file exsits",
			ldconfig:    "@/does/not/exist",
			expected:    "@/does/not/exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c := ContainerCLIConfig{
				Ldconfig: tc.ldconfig,
			}

			require.Equal(t, tc.expected, c.NormalizeLDConfigPath())
		})
	}
}
