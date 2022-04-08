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
		spec, err := LoadFrom(bytes.NewReader(tc.contents))

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
