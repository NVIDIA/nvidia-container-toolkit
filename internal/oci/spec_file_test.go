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

func TestLookupEnv(t *testing.T) {
	const envName = "TEST_ENV"
	testCases := []struct {
		spec          *specs.Spec
		expectedValue string
		expectedExits bool
	}{
		{
			// nil spec
			spec:          nil,
			expectedValue: "",
			expectedExits: false,
		},
		{
			// nil process
			spec:          &specs.Spec{},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// nil env
			spec: &specs.Spec{
				Process: &specs.Process{},
			},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// empty env
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{}},
			},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// different env set
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"SOMETHING_ELSE=foo"}},
			},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// same prefix
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"TEST_ENV_BUT_NOT=foo"}},
			},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// same suffix
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"NOT_TEST_ENV=foo"}},
			},
			expectedValue: "",
			expectedExits: false,
		},
		{
			// set blank
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"TEST_ENV="}},
			},
			expectedValue: "",
			expectedExits: true,
		},
		{
			// set no-equals
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"TEST_ENV"}},
			},
			expectedValue: "",
			expectedExits: true,
		},
		{
			// set value
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"TEST_ENV=something"}},
			},
			expectedValue: "something",
			expectedExits: true,
		},
		{
			// set with equals
			spec: &specs.Spec{
				Process: &specs.Process{Env: []string{"TEST_ENV=something=somethingelse"}},
			},
			expectedValue: "something=somethingelse",
			expectedExits: true,
		},
	}

	for i, tc := range testCases {
		spec := fileSpec{
			Spec: tc.spec,
		}

		value, exists := spec.LookupEnv(envName)

		require.Equal(t, tc.expectedValue, value, "%d: %v", i, tc)
		require.Equal(t, tc.expectedExits, exists, "%d: %v", i, tc)
	}
}

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
		spec := fileSpec{}
		err := spec.loadFrom(bytes.NewReader(tc.contents))

		if tc.isError {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}

		if tc.spec == nil {
			require.Nil(t, spec.Spec, "%d: %v", i, tc)
		} else {
			require.EqualValues(t, tc.spec, spec.Spec, "%d: %v", i, tc)
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

		spec := fileSpec{Spec: tc.spec}
		err := spec.flushTo(&buffer)

		if tc.isError {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}

		require.EqualValues(t, tc.contents, buffer.String(), "%d: %v", i, tc)
	}

	// Add a simple test for a writer that returns an error when writing
	spec := fileSpec{Spec: &specs.Spec{}}
	err := spec.flushTo(errorWriter{})
	require.Error(t, err)
}

func TestModify(t *testing.T) {

	testCases := []struct {
		spec          *specs.Spec
		modifierError error
	}{
		{
			spec: nil,
		},
		{
			spec: &specs.Spec{},
		},
		{
			spec:          &specs.Spec{},
			modifierError: fmt.Errorf("error in modifier"),
		},
	}

	for i, tc := range testCases {
		spec := fileSpec{Spec: tc.spec}

		modifier := func(spec *specs.Spec) error {
			if tc.modifierError == nil {
				spec.Version = "updated"
			}
			return tc.modifierError
		}

		err := spec.Modify(modifier)

		if tc.spec == nil {
			require.Error(t, err, "%d: %v", i, tc)
		} else if tc.modifierError != nil {
			require.EqualError(t, err, tc.modifierError.Error(), "%d: %v", i, tc)
			require.EqualValues(t, &specs.Spec{}, spec.Spec, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
			require.Equal(t, "updated", spec.Spec.Version, "%d: %v", i, tc)
		}
	}
}

// errorWriter implements the io.Writer interface, always returning an error when
// writing.
type errorWriter struct{}

func (e errorWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("error writing")
}
