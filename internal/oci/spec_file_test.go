/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package oci

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestLoadFrom(t *testing.T) {
	testCases := []struct {
		contents []byte
		isError  bool
		spec     *specs.Spec
	}{
		{
			contents: []byte{},
			isError:  true,
		},
		{
			contents: []byte("{}"),
			isError:  false,
			spec:     &specs.Spec{},
		},
	}

	for i, tc := range testCases {
		var spec *specs.Spec
		spec, err := strictLoader.loadFrom(bytes.NewReader(tc.contents))

		if tc.isError {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}

		if tc.spec == nil {
			require.Nil(t, spec, "%d: %v", i, tc)
		} else {
			require.EqualValues(t, tc.spec, spec, "%d: %v", i, tc)
		}
	}
}

func TestFlushTo(t *testing.T) {
	testCases := []struct {
		isError  bool
		spec     *specs.Spec
		contents string
	}{
		{
			spec: nil,
		},
		{
			spec:     &specs.Spec{},
			contents: "{\"ociVersion\":\"\"}\n",
		},
	}

	for i, tc := range testCases {
		buffer := bytes.Buffer{}

		err := flushTo(tc.spec, &buffer)

		if tc.isError {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}

		require.EqualValues(t, tc.contents, buffer.String(), "%d: %v", i, tc)
	}

	// Add a simple test for a writer that returns an error when writing
	err := flushTo(&specs.Spec{}, errorWriter{})
	require.Error(t, err)
}

// errorWriter implements the io.Writer interface, always returning an error when
// writing.
type errorWriter struct{}

func (e errorWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("error writing")
}

// TestFlushPreservesOriginalOnWriteFailure asserts that when Flush cannot write
// (e.g. the target directory is read-only), the original OCI spec file is left
// untouched instead of being truncated and lost.
func TestFlushPreservesOriginalOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, "config.json")
	originalContent := `{"ociVersion": ""}`
	require.NoError(t, os.WriteFile(originalPath, []byte(originalContent), 0o644))

	// Make the directory read-only so the temp file cannot be created. The
	// original file must remain intact.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	spec := &fileSpec{path: originalPath, loader: strictLoader, Spec: &specs.Spec{}}

	err := spec.Flush()
	require.Error(t, err)

	contents, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	require.Equal(t, originalContent, string(contents))
}

// TestFlushPreservesFileMode asserts that Flush preserves the mode of the
// original file on the replacement, so a 0444 config.json does not become
// 0600 after a successful atomic replace.
func TestFlushPreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(originalPath, []byte(`{"ociVersion": ""}`), 0o444))

	spec := &fileSpec{path: originalPath, loader: strictLoader, Spec: &specs.Spec{}}
	require.NoError(t, spec.Flush())

	t.Cleanup(func() { os.Chmod(originalPath, 0o644) })

	info, err := os.Stat(originalPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o444), info.Mode().Perm())
}

// TestFlushFollowsSymlink asserts that Flush writes through a symlink to its
// target, preserving the symlink itself (matching the os.Create semantics of
// following symlinks rather than replacing them).
func TestFlushFollowsSymlink(t *testing.T) {
	dir := t.TempDir()

	// Create a real target file with known content.
	targetPath := filepath.Join(dir, "config.target.json")
	originalContent := `{"ociVersion": ""}`
	require.NoError(t, os.WriteFile(targetPath, []byte(originalContent), 0o644))

	// Create a symlink pointing to the target.
	linkPath := filepath.Join(dir, "config.link.json")
	require.NoError(t, os.Symlink(targetPath, linkPath))

	spec := &fileSpec{path: linkPath, loader: strictLoader, Spec: &specs.Spec{}}
	require.NoError(t, spec.Flush())

	// The symlink should still exist and not have been replaced by a regular file.
	linkInfo, err := os.Lstat(linkPath)
	require.NoError(t, err)
	require.True(t, linkInfo.Mode()&os.ModeSymlink != 0, "symlink should not be replaced by Flush")

	// The target content should have been updated with the new spec.
	targetContent, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Contains(t, string(targetContent), "ociVersion")

	// Reading through the symlink should also reflect the updated content.
	linkContent, err := os.ReadFile(linkPath)
	require.NoError(t, err)
	require.Equal(t, string(targetContent), string(linkContent))
}
