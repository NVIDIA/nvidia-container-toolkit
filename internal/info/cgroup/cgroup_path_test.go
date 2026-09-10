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
**/

package cgroup

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAbsolutePath(t *testing.T) {
	_, err := GetAbsolutePath(math.MaxInt32)
	require.Error(t, err)
}

func TestParseCgroupProcFile(t *testing.T) {
	testCases := []struct {
		description string
		contents    string
		expected    string
		expectError bool
	}{
		{
			description: "cgroup v2 unified hierarchy",
			contents:    "0::/user.slice/user-1000.slice/session-1.scope\n",
			expected:    filepath.Join(rootDirectory, "/user.slice/user-1000.slice/session-1.scope"),
		},
		{
			description: "cgroup v1 named hierarchy",
			contents:    "5:devices:/docker/abc123\n",
			expected:    filepath.Join(rootDirectory, "/docker/abc123"),
		},
		{
			description: "root cgroup path",
			contents:    "0::/\n",
			expected:    filepath.Join(rootDirectory, "/"),
		},
		{
			description: "missing trailing newline",
			contents:    "0::/foo",
			expected:    filepath.Join(rootDirectory, "/foo"),
		},
		{
			description: "empty contents",
			contents:    "",
			expectError: true,
		},
		{
			description: "too few fields",
			contents:    "0:/foo",
			expectError: true,
		},
		{
			description: "too many fields",
			contents:    "0::/foo:extra",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			path, err := parseCgroupProcFile([]byte(tc.contents))
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, path)
		})
	}
}
