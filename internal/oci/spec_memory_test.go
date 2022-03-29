/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
		spec := memorySpec{
			Spec: tc.spec,
		}

		value, exists := spec.LookupEnv(envName)

		require.Equal(t, tc.expectedValue, value, "%d: %v", i, tc)
		require.Equal(t, tc.expectedExits, exists, "%d: %v", i, tc)
	}
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
		spec := NewMemorySpec(tc.spec).(*memorySpec)

		err := spec.Modify(modifier{tc.modifierError})

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

// TODO: Ideally we would generated a mock for the SpecModifer too. This causes
// an import cycle and we define a local type as a work-around.
type modifier struct {
	modifierError error
}

func (m modifier) Modify(spec *specs.Spec) error {
	if m.modifierError == nil {
		spec.Version = "updated"
	}
	return m.modifierError
}
