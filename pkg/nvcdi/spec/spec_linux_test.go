/**
# Copyright 2024 NVIDIA CORPORATION
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

package spec

import (
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"
)

func TestWriteToClosesTemporaryFile(t *testing.T) {
	// A file finalizer must not hide a missing Close on the error path.
	previousGCPercent := debug.SetGCPercent(-1)
	t.Cleanup(func() { debug.SetGCPercent(previousGCPercent) })

	testCases := []struct {
		description   string
		version       string
		expectedError string
	}{
		{description: "save succeeds", version: "0.3.0"},
		{description: "save fails validation", version: "invalid", expectedError: `invalid version "invalid"`},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("TMPDIR", tempDir)
			s, err := New(WithRawSpec(&specs.Spec{
				Version: tc.version,
				Kind:    "nvidia.com/gpu",
				Devices: []specs.Device{{
					Name:           "one",
					ContainerEdits: specs.ContainerEdits{Env: []string{"DEVICE_FOO=bar"}},
				}},
			}))
			require.NoError(t, err)

			n, err := s.WriteTo(io.Discard)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				require.Zero(t, n)
			} else {
				require.NoError(t, err)
				require.Positive(t, n)
			}

			files, err := os.ReadDir(tempDir)
			require.NoError(t, err)
			require.Empty(t, files, "temporary files should be removed")

			descriptors, err := os.ReadDir("/proc/self/fd")
			require.NoError(t, err)
			var openTemporaryFiles []string
			for _, descriptor := range descriptors {
				target, err := os.Readlink(filepath.Join("/proc/self/fd", descriptor.Name()))
				if err == nil && strings.HasPrefix(target, tempDir+string(os.PathSeparator)) {
					openTemporaryFiles = append(openTemporaryFiles, target)
				}
			}
			require.Empty(t, openTemporaryFiles, "temporary file descriptors should be closed")
		})
	}
}
