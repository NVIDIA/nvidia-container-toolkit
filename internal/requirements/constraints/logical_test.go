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

package constraints

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestANDConstraint(t *testing.T) {

	never := ConstraintMock{AssertFunc: func() error { return fmt.Errorf("false") }}

	testCases := []struct {
		description string
		constraints []Constraint
		expected    bool
	}{
		{
			description: "empty is always true",
			constraints: []Constraint{},
			expected:    true,
		},
		{
			description: "single true constraint is true",
			constraints: []Constraint{
				&always{},
			},
			expected: true,
		},
		{
			description: "single false constraint is false",
			constraints: []Constraint{
				&never,
			},
			expected: false,
		},
		{
			description: "multiple true constraints are true",
			constraints: []Constraint{
				&always{}, &always{},
			},
			expected: true,
		},
		{
			description: "mixed constraints are false (first is true)",
			constraints: []Constraint{
				&always{}, &never,
			},
			expected: false,
		},
		{
			description: "mixed constraints are false (last is true)",
			constraints: []Constraint{
				&never, &always{},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := and(tc.constraints).Assert()
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestORConstraint(t *testing.T) {

	never := ConstraintMock{AssertFunc: func() error { return fmt.Errorf("false") }}

	testCases := []struct {
		description string
		constraints []Constraint
		expected    bool
	}{
		{
			description: "empty is always false",
			constraints: []Constraint{},
			expected:    false,
		},
		{
			description: "single true constraint is true",
			constraints: []Constraint{
				&always{},
			},
			expected: true,
		},
		{
			description: "single false constraint is false",
			constraints: []Constraint{
				&never,
			},
			expected: false,
		},
		{
			description: "multiple true constraints are true",
			constraints: []Constraint{
				&always{}, &always{},
			},
			expected: true,
		},
		{
			description: "mixed constraints are true (first is true)",
			constraints: []Constraint{
				&always{}, &never,
			},
			expected: true,
		},
		{
			description: "mixed constraints are true (last is true)",
			constraints: []Constraint{
				&never, &always{},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := or(tc.constraints).Assert()
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

}
