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

package root

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestDriverLibrariesLocate(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	testCases := []struct {
		rootFs        string
		inputs        []string
		expected      string
		expectedError error
	}{
		{
			rootFs:        "rootfs-empty",
			inputs:        []string{"libcuda.so.1", "libcuda.so.*", "libcuda.so.*.*", "libcuda.so.999.88.77"},
			expectedError: lookup.ErrNotFound,
		},
		{
			rootFs:   "rootfs-no-cache-lib64",
			inputs:   []string{"libcuda.so.1", "libcuda.so.*", "libcuda.so.*.*", "libcuda.so.999.88.77"},
			expected: "/usr/lib64/libcuda.so.999.88.77",
		},
		{
			rootFs:   "rootfs-1",
			inputs:   []string{"libcuda.so.1", "libcuda.so.*", "libcuda.so.*.*", "libcuda.so.999.88.77"},
			expected: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77",
		},
		{
			rootFs:   "rootfs-2",
			inputs:   []string{"libcuda.so.1", "libcuda.so.*", "libcuda.so.*.*", "libcuda.so.999.88.77"},
			expected: "/var/lib/nvidia/lib64/libcuda.so.999.88.77",
		},
	}

	for _, tc := range testCases {
		for _, input := range tc.inputs {
			t.Run(tc.rootFs+input, func(t *testing.T) {
				rootfs := filepath.Join(moduleRoot, "testdata", "lookup", tc.rootFs)
				driver := New(
					WithLogger(logger),
					WithDriverRoot(rootfs),
				)

				candidates, err := driver.Libraries().Locate(input)
				require.ErrorIs(t, err, tc.expectedError)
				if tc.expectedError == nil {
					require.Equal(t, []string{filepath.Join(rootfs, tc.expected)}, candidates)
				}
			})
		}
	}
}
