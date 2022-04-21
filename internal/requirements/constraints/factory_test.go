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
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	cuda := NewVersionProperty("cuda", "")

	f := factory{
		logger: logger,
		properties: map[string]Property{
			"cuda": cuda,
		},
	}

	testCases := []struct {
		description   string
		condition     string
		expectedError bool
		expected      Constraint
	}{
		{
			description: "empty is nil",
			condition:   "",
			expected:    nil,
		},
		{
			description:   "missing operator is invalid",
			condition:     "notvalid",
			expectedError: true,
		},
		{
			description: "invalid property is invalid",
			condition:   "foo=45",
			expected:    nil,
		},
		{
			description:   "cuda must be semver",
			condition:     "cuda=foo",
			expectedError: true,
			expected:      binary{cuda, equal, "foo"},
		},
		{
			description: "cuda greater than equal",
			condition:   "cuda>=11.6",
			expected:    binary{cuda, greaterEqual, "11.6"},
		},
		{
			description: "cuda greater than",
			condition:   "cuda>11.6",
			expected:    binary{cuda, greater, "11.6"},
		},
		{
			description: "cuda less than equal",
			condition:   "cuda<=11.6",
			expected:    binary{cuda, lessEqual, "11.6"},
		},
		{
			description: "cuda less than",
			condition:   "cuda<11.6",
			expected:    binary{cuda, less, "11.6"},
		},
		{
			description: "cuda equal",
			condition:   "cuda=11.6",
			expected:    binary{cuda, equal, "11.6"},
		},
		{
			description: "cuda not equal",
			condition:   "cuda!=11.6",
			expected:    binary{cuda, notEqual, "11.6"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c, err := f.parse(tc.condition)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.EqualValues(t, tc.expected, c)
		})
	}
}

func TestNewConstraintFromRequirement(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	cuda := &PropertyMock{}
	arch := &PropertyMock{}

	f := factory{
		logger: logger,
		properties: map[string]Property{
			"cuda": cuda,
			"arch": arch,
		},
	}

	testCases := []struct {
		description   string
		requirement   string
		expectedError bool
		expected      Constraint
	}{
		{
			description: "empty is nil",
			requirement: "",
			expected:    nil,
		},
		{
			description:   "malformed constraint is invalid",
			requirement:   "notvalid",
			expectedError: true,
			expected:      nil,
		},
		{
			description: "unsupported property is ignored",
			requirement: "cuda>=11.6 foo=bar",
			expected:    binary{cuda, greaterEqual, "11.6"},
		},
		{
			description: "space-separated is and",
			requirement: "cuda>=11.6 arch=5.3",
			expected: and([]Constraint{
				binary{cuda, greaterEqual, "11.6"},
				binary{arch, equal, "5.3"},
			}),
		},
		{
			description: "comma-separated is or",
			requirement: "cuda>=11.6,arch=5.3",
			expected: or([]Constraint{
				binary{cuda, greaterEqual, "11.6"},
				binary{arch, equal, "5.3"},
			}),
		},
		{
			description: "and takes precedence",
			requirement: "cuda<13.6 cuda>=11.6,arch=5.3",
			expected: or([]Constraint{
				binary{cuda, less, "13.6"},
				and([]Constraint{
					binary{cuda, greaterEqual, "11.6"},
					binary{arch, equal, "5.3"},
				}),
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c, err := f.newConstraintFromRequirement(tc.requirement)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.EqualValues(t, tc.expected, c)
		})
	}

}
