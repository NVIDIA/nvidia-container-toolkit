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

func TestFileSpecFlush(t *testing.T) {
	testCases := []struct {
		name     string
		contents string
		mode     os.FileMode
	}{
		{
			name:     "existing file",
			contents: `{"ociVersion":"1.0.0"}`,
			mode:     0600,
		},
		{
			name:     "existing file with default mode",
			contents: `{"ociVersion":"1.0.0"}`,
			mode:     0644,
		},
		{
			name:     "new file",
			contents: "",
			mode:     0644,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			if tc.contents != "" {
				require.NoError(t, os.WriteFile(path, []byte(tc.contents), tc.mode))
			}

			spec := NewFileSpec(path, true).(*fileSpec)
			if tc.contents != "" {
				_, err := spec.Load()
				require.NoError(t, err)
			} else {
				spec.Spec = &specs.Spec{Version: "1.0.0"}
			}

			require.NoError(t, spec.Modify(specModifierFunc(func(s *specs.Spec) error {
				if s.Annotations == nil {
					s.Annotations = map[string]string{}
				}
				s.Annotations["test"] = "annotation"
				return nil
			})))

			require.NoError(t, spec.Flush())

			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, `{"ociVersion":"1.0.0","annotations":{"test":"annotation"}}`, string(contents))

			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, tc.mode, info.Mode().Perm())

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1, "temporary file should not be left behind")
		})
	}
}

// TestFileSpecFlushFailurePreservesFile is a regression test: a failed Flush must not
// truncate or otherwise modify an existing spec file.
func TestFileSpecFlushFailurePreservesFile(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0755)
	})

	path := filepath.Join(dir, "config.json")
	contents := `{"ociVersion":"1.0.0"}`
	require.NoError(t, os.WriteFile(path, []byte(contents), 0644))

	// Make the directory read-only so that creating the temporary file fails. The
	// existing file itself remains writable, which is what allowed the old
	// implementation to truncate it before the write failed.
	require.NoError(t, os.Chmod(dir, 0555))

	spec := NewFileSpec(path, true)
	_, err := spec.Load()
	require.NoError(t, err)

	require.Error(t, spec.Flush())

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, contents, string(actual))
}

// specModifierFunc adapts a function to the SpecModifier interface.
type specModifierFunc func(*specs.Spec) error

func (f specModifierFunc) Modify(s *specs.Spec) error {
	return f(s)
}

func TestFileSpecFlushWithoutLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	contents := `{"ociVersion":"1.0.0"}`
	require.NoError(t, os.WriteFile(path, []byte(contents), 0644))

	spec := NewFileSpec(path, true)
	require.Error(t, spec.Flush())

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, contents, string(actual))
}
