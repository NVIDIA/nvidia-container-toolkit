/**
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
**/

package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateLdconfig(t *testing.T) {
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
			c := ContainerRoot(containerRootDir)
			err := c.CreateLdsoconfdFile("00-test-*.conf", tc.folders...)
			require.NoError(t, err)

			matches, err := filepath.Glob(filepath.Join(containerRootDir, "/etc/ld.so.conf.d/00-test-*.conf"))
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
