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

package cuda

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

func TestLocate(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		libcudaPath   string
		expected      []string
		expectedError error
	}{
		{
			description:   "no libcuda does not resolve library",
			libcudaPath:   "",
			expected:      []string{},
			expectedError: lookup.ErrNotFound,
		},
		{
			description:   "no-ldcache searches /usr/lib64",
			libcudaPath:   "/usr/lib64/libcuda.so.123.34",
			expected:      []string{"/usr/lib64/libcuda.so.123.34"},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			driverRoot, err := setupDriverRoot(t, tc.libcudaPath)
			require.NoError(t, err)

			l := New(
				lookup.NewLibraryLocator(
					lookup.WithLogger(logger),
					lookup.WithRoot(driverRoot),
				),
			)

			candidates, err := l.Locate(".*")
			require.ErrorIs(t, err, tc.expectedError)

			var strippedCandidates []string
			for _, c := range candidates {
				// NOTE: We need to strip `/private` on MacOs due to symlink resolution
				strippedCandidates = append(strippedCandidates, strings.TrimPrefix(c, "/private"))
			}
			var expectedWithRoot []string
			for _, e := range tc.expected {
				expectedWithRoot = append(expectedWithRoot, filepath.Join(driverRoot, e))
			}

			require.EqualValues(t, expectedWithRoot, strippedCandidates)
		})
	}
}

// setupDriverRoot creates a folder that can be used to represent a driver root.
// The path to libcuda can be specified and an empty file is created at this location in the driver root.
func setupDriverRoot(t *testing.T, libCudaPath string) (string, error) {
	driverRoot := t.TempDir()

	if libCudaPath == "" {
		return driverRoot, nil
	}

	if err := os.MkdirAll(filepath.Join(driverRoot, filepath.Dir(libCudaPath)), 0755); err != nil {
		return "", fmt.Errorf("falied to create required driver root folder: %w", err)
	}

	libCuda, err := os.Create(filepath.Join(driverRoot, libCudaPath))
	if err != nil {
		return "", fmt.Errorf("failed to create dummy libcuda.so: %w", err)
	}
	defer libCuda.Close()

	return driverRoot, nil
}
