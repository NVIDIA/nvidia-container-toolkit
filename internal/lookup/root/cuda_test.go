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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

func TestLocate(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		libPaths      []string
		expected      []string
		expectedError error
	}{
		{
			description:   "no libcuda does not resolve library",
			libPaths:      nil,
			expected:      nil,
			expectedError: lookup.ErrNotFound,
		},
		{
			description:   "no-ldcache searches /usr/lib64",
			libPaths:      []string{"/usr/lib64/libcuda.so.123.34"},
			expected:      []string{"/usr/lib64"},
			expectedError: nil,
		},
		{
			description:   "no-ldcache searches /usr/lib64 for libnvidia-ml.so.",
			libPaths:      []string{"/usr/lib64/libnvidia-ml.so.123.34"},
			expected:      []string{"/usr/lib64"},
			expectedError: nil,
		},
		{
			description: "locates two driver library directories",
			libPaths: []string{
				"/usr/lib64/libcuda.so.123.34",
				"/usr/lib/x86_64-linux-gnu/libnvidia-ml.so.123.34",
			},
			expected: []string{
				"/usr/lib64",
				"/usr/lib/x86_64-linux-gnu",
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			driverRoot, err := setupDriverRoot(t, tc.libPaths)
			require.NoError(t, err)

			l := New(
				WithLogger(logger),
				WithDriverRoot(driverRoot),
			)

			driverLibraryPaths, err := l.GetDriverLibDirectories()
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			// NOTE: We need to strip `/private` on MacOs due to symlink resolution
			stripped := make([]string, len(driverLibraryPaths))
			for i, p := range driverLibraryPaths {
				stripped[i] = strings.TrimPrefix(p, "/private")
			}

			require.ElementsMatch(t, tc.expected, stripped)
		})
	}
}

// setupDriverRoot creates a folder that can be used to represent a driver root.
// Library paths can be specified and empty files are created at these locations in the driver root.
func setupDriverRoot(t *testing.T, libPaths []string) (string, error) {
	driverRoot := t.TempDir()

	for _, libPath := range libPaths {
		if err := os.MkdirAll(filepath.Join(driverRoot, filepath.Dir(libPath)), 0755); err != nil {
			return "", fmt.Errorf("failed to create required driver root folder: %w", err)
		}

		f, err := os.Create(filepath.Join(driverRoot, libPath))
		if err != nil {
			return "", fmt.Errorf("failed to create dummy library file: %w", err)
		}
		f.Close()
	}

	return filepath.EvalSymlinks(driverRoot)
}
