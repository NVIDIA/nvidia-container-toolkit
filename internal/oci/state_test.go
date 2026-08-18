/**
# Copyright (c), NVIDIA CORPORATION.  All rights reserved.
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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestReadContainerState(t *testing.T) {
	testCases := []struct {
		description string
		contents    string
		isError     bool
		expected    *State
	}{
		{
			description: "invalid json returns error",
			contents:    "not json",
			isError:     true,
		},
		{
			description: "empty object decodes to empty state",
			contents:    "{}",
			expected:    &State{},
		},
		{
			description: "standard fields are decoded",
			contents:    `{"bundle": "/foo/bar"}`,
			expected: &State{
				State: specs.State{
					Bundle: "/foo/bar",
				},
			},
		},
		{
			description: "non-standard root extension is decoded",
			contents:    `{"bundle": "/foo/bar", "root": "/foo/bar/rootfs"}`,
			expected: &State{
				State: specs.State{
					Bundle: "/foo/bar",
				},
				Root: "/foo/bar/rootfs",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			s, err := ReadContainerState(bytes.NewBufferString(tc.contents))

			if tc.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, tc.expected, s)
		})
	}
}

func TestLoadContainerState(t *testing.T) {
	testCases := []struct {
		description    string
		useStdin       bool
		useMissingFile bool
		stateJSON      string
		isError        bool
		expectedBundle string
	}{
		{
			description:    "reads from stdin when filename is empty",
			useStdin:       true,
			stateJSON:      `{"bundle": "/from/stdin"}`,
			expectedBundle: "/from/stdin",
		},
		{
			description:    "reads from a file when filename is specified",
			stateJSON:      `{"bundle": "/from/file"}`,
			expectedBundle: "/from/file",
		},
		{
			description:    "returns an error when the file does not exist",
			useMissingFile: true,
			isError:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var filename string
			switch {
			case tc.useStdin:
				oldStdin := os.Stdin
				r, w, err := os.Pipe()
				require.NoError(t, err)
				_, err = w.WriteString(tc.stateJSON)
				require.NoError(t, err)
				require.NoError(t, w.Close())
				os.Stdin = r
				t.Cleanup(func() { os.Stdin = oldStdin })
			case tc.useMissingFile:
				filename = filepath.Join(t.TempDir(), "does-not-exist.json")
			default:
				filename = filepath.Join(t.TempDir(), "state.json")
				require.NoError(t, os.WriteFile(filename, []byte(tc.stateJSON), 0600))
			}

			s, err := LoadContainerState(filename)

			if tc.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedBundle, s.Bundle)
		})
	}
}

func TestGetContainerRoot(t *testing.T) {
	testCases := []struct {
		description  string
		root         string
		specJSON     string
		writeSpec    bool
		isError      bool
		expectedRoot func(bundle string) string
	}{
		{
			description: "absolute root extension is returned as-is",
			root:        "/absolute/rootfs",
			expectedRoot: func(bundle string) string {
				return "/absolute/rootfs"
			},
		},
		{
			description: "relative root extension is joined with the bundle",
			root:        "rootfs",
			expectedRoot: func(bundle string) string {
				return filepath.Join(bundle, "rootfs")
			},
		},
		{
			description: "falls back to the spec file when root extension is not set",
			writeSpec:   true,
			specJSON:    `{"root": {"path": "rootfs"}}`,
			expectedRoot: func(bundle string) string {
				return filepath.Join(bundle, "rootfs")
			},
		},
		{
			description: "returns an empty string when neither root extension nor spec root are set",
			writeSpec:   true,
			specJSON:    `{}`,
			expectedRoot: func(bundle string) string {
				return bundle
			},
		},
		{
			description: "returns an error when the spec file cannot be loaded",
			writeSpec:   false,
			isError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dir := t.TempDir()
			if tc.writeSpec {
				require.NoError(t, os.WriteFile(GetSpecFilePath(dir), []byte(tc.specJSON), 0600))
			}
			s := &State{
				State: specs.State{Bundle: dir},
				Root:  tc.root,
			}

			root, err := s.GetContainerRoot()

			if tc.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedRoot(dir), root)
		})
	}
}

func TestLoadMinimalSpec(t *testing.T) {
	testCases := []struct {
		description  string
		specJSON     string
		writeSpec    bool
		isError      bool
		expectedRoot string
	}{
		{
			description: "returns an error when the spec file does not exist",
			writeSpec:   false,
			isError:     true,
		},
		{
			description: "returns an error for invalid json",
			writeSpec:   true,
			specJSON:    "not json",
			isError:     true,
		},
		{
			description:  "decodes the root field",
			writeSpec:    true,
			specJSON:     `{"root": {"path": "/some/rootfs"}}`,
			expectedRoot: "/some/rootfs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dir := t.TempDir()
			if tc.writeSpec {
				require.NoError(t, os.WriteFile(GetSpecFilePath(dir), []byte(tc.specJSON), 0600))
			}
			s := &State{State: specs.State{Bundle: dir}}

			ms, err := s.loadMinimalSpec()

			if tc.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedRoot, ms.Root.Path)
		})
	}
}
