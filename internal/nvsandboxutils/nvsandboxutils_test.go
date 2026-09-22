/**
# Copyright 2026 NVIDIA CORPORATION
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

package nvsandboxutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	mockRootfs        = "/mock/rootfs"
	mockGpuUUID       = "GPU-6fd91e39-6ba7-4d6b-b1d6-1dffb0bc7a2f"
	mockMigUUID       = "MIG-6fd91e39-6ba7-4d6b-b1d6-1dffb0bc7a2f"
	mockDriverVersion = "999.88.77"

	mockSymlinkPath   = "/dev/dri/by-path/pci-0000:41:00.0-card"
	mockSymlinkTarget = "../card1"
)

func buildMockLibrary(t *testing.T) string {
	t.Helper()

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	if _, err := exec.LookPath(cc); err != nil {
		t.Skipf("no C compiler available: %v", err)
	}

	path := filepath.Join(t.TempDir(), "mock_nvsandbox.so")
	cmd := exec.Command(cc, "-shared", "-fPIC", "-I", ".", "-o", path, filepath.Join("testdata", "mock_nvsandbox.c"))
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "failed to build mock library: %s", output)

	return path
}

func newMockLibrary(t *testing.T) *library {
	t.Helper()

	l := newLibrary(WithLibraryPath(buildMockLibrary(t)))
	t.Cleanup(func() { _ = l.close() })

	return l
}

func newInitializedMockLibrary(t *testing.T) *library {
	t.Helper()

	l := newMockLibrary(t)
	require.Equal(t, SUCCESS, l.Init(mockRootfs))

	return l
}

func TestEntryPointBeforeLoadReturnsLibraryLoadError(t *testing.T) {
	require.Equal(t, ERROR_LIBRARY_LOAD, nvSandboxUtilsShutdown())
}

func TestInitRejectsAnUnexpectedRootfs(t *testing.T) {
	l := newMockLibrary(t)

	require.Equal(t, ERROR_INVALID_ARG, l.Init("/not/the/mock/rootfs"))
}

func TestGetDriverVersion(t *testing.T) {
	l := newInitializedMockLibrary(t)

	version, ret := l.GetDriverVersion()
	require.Equal(t, SUCCESS, ret)
	require.Equal(t, mockDriverVersion, version)
}

func TestGetFileContent(t *testing.T) {
	testCases := []struct {
		description     string
		path            string
		expectedContent string
		expectedRet     Ret
	}{
		{
			description:     "known path",
			path:            mockSymlinkPath,
			expectedContent: mockSymlinkTarget,
			expectedRet:     SUCCESS,
		},
		{
			description: "unknown path",
			path:        "/not/a/known/path",
			expectedRet: ERROR_FILEPATH_NOT_FOUND,
		},
	}

	l := newInitializedMockLibrary(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			content, ret := l.GetFileContent(tc.path)
			require.Equal(t, tc.expectedRet, ret)
			if tc.expectedRet == SUCCESS {
				require.Equal(t, tc.expectedContent, content)
			}
		})
	}
}

func TestGetGpuResource(t *testing.T) {
	testCases := []struct {
		description string
		uuid        string
		expected    []GpuFileInfo
		expectedRet Ret
	}{
		{
			description: "GPU UUID returns the full list",
			uuid:        mockGpuUUID,
			expected: []GpuFileInfo{
				{
					Path:    "/dev/nvidia0",
					Type:    NV_DEV,
					SubType: NV_DEV_NVIDIA,
					Module:  NV_GPU,
					Flags:   NV_FILE_FLAG_HINT,
				},
				{
					Path:    "/dev/nvidiactl",
					Type:    NV_DEV,
					SubType: NV_DEV_NVIDIA_CTL,
					Module:  NV_DRIVER_NVIDIA,
					Flags:   NV_FILE_FLAG_HINT,
				},
			},
			expectedRet: SUCCESS,
		},
		{
			description: "MIG UUID selects the MIG input type",
			uuid:        mockMigUUID,
			expected: []GpuFileInfo{
				{
					Path:    "/dev/nvidiactl",
					Type:    NV_DEV,
					SubType: NV_DEV_NVIDIA_CTL,
					Module:  NV_DRIVER_NVIDIA,
					Flags:   NV_FILE_FLAG_HINT,
				},
			},
			expectedRet: SUCCESS,
		},
		{
			description: "unknown UUID",
			uuid:        "GPU-unknown",
			expectedRet: ERROR_DEVICE_NOT_FOUND,
		},
	}

	l := newInitializedMockLibrary(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			files, ret := l.GetGpuResource(tc.uuid)
			require.Equal(t, tc.expectedRet, ret)
			require.Equal(t, tc.expected, files)
		})
	}
}

func TestShutdownUnloadsTheLibrary(t *testing.T) {
	l := newInitializedMockLibrary(t)

	require.Equal(t, SUCCESS, l.Shutdown())
	require.Equal(t, refcount(0), l.refcount)
}
