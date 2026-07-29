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

package root

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformPath(t *testing.T) {
	testCases := []struct {
		description  string
		root         string
		targetRoot   string
		path         string
		expectedPath string
	}{
		{
			description:  "path under root is transformed",
			root:         "/run/nvidia/driver",
			targetRoot:   "/",
			path:         "/run/nvidia/driver/lib/libcuda.so",
			expectedPath: "/lib/libcuda.so",
		},
		{
			description:  "path equal to root is transformed",
			root:         "/run/nvidia/driver",
			targetRoot:   "/host",
			path:         "/run/nvidia/driver",
			expectedPath: "/host",
		},
		{
			description:  "path outside root is unchanged",
			root:         "/run/nvidia/driver",
			targetRoot:   "/host",
			path:         "/usr/lib/libcuda.so",
			expectedPath: "/usr/lib/libcuda.so",
		},
		{
			description:  "sibling of root with common prefix is unchanged",
			root:         "/run/nvidia/driver",
			targetRoot:   "/host",
			path:         "/run/nvidia/driver-backup/lib/libcuda.so",
			expectedPath: "/run/nvidia/driver-backup/lib/libcuda.so",
		},
		{
			description:  "partial path component match is unchanged",
			root:         "/run/nvidia/driver",
			targetRoot:   "/host",
			path:         "/run/nvidia/driverfoo",
			expectedPath: "/run/nvidia/driverfoo",
		},
		{
			description:  "root with trailing slash is transformed",
			root:         "/run/nvidia/driver/",
			targetRoot:   "/host",
			path:         "/run/nvidia/driver/lib/libcuda.so",
			expectedPath: "/host/lib/libcuda.so",
		},
		{
			description:  "root of / is transformed",
			root:         "/",
			targetRoot:   "/host",
			path:         "/usr/lib/libcuda.so",
			expectedPath: "/host/usr/lib/libcuda.so",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tr := transformer{
				root:       tc.root,
				targetRoot: tc.targetRoot,
			}
			require.Equal(t, tc.expectedPath, tr.transformPath(tc.path))
		})
	}
}
