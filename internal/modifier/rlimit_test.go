/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package modifier

import (
	"math"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
)

func TestParseRlimits(t *testing.T) {
	testCases := []struct {
		description   string
		entries       []string
		expected      []specs.POSIXRlimit
		expectedError bool
	}{
		{
			description: "empty entries yield no rlimits",
		},
		{
			description: "unlimited value applies to soft and hard",
			entries:     []string{"memlock=unlimited"},
			expected: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
			},
		},
		{
			description: "infinity is accepted as an alias",
			entries:     []string{"memlock=infinity"},
			expected: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
			},
		},
		{
			description: "RLIMIT_ prefix and case are normalized",
			entries:     []string{"RLIMIT_MEMLOCK=unlimited", "NoFile=1024:2048"},
			expected: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
				{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 2048},
			},
		},
		{
			description: "numeric soft and hard values",
			entries:     []string{"memlock=1073741824:2147483648"},
			expected: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: 1073741824, Hard: 2147483648},
			},
		},
		{
			description: "numeric soft with unlimited hard",
			entries:     []string{"memlock=1073741824:unlimited"},
			expected: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: 1073741824, Hard: math.MaxUint64},
			},
		},
		{
			description:   "missing separator is rejected",
			entries:       []string{"memlock"},
			expectedError: true,
		},
		{
			description:   "empty name is rejected",
			entries:       []string{"=unlimited"},
			expectedError: true,
		},
		{
			description:   "non-numeric value is rejected",
			entries:       []string{"memlock=lots"},
			expectedError: true,
		},
		{
			description:   "negative value is rejected",
			entries:       []string{"memlock=-1"},
			expectedError: true,
		},
		{
			description:   "soft limit above hard limit is rejected",
			entries:       []string{"memlock=2048:1024"},
			expectedError: true,
		},
		{
			description:   "duplicate types are rejected",
			entries:       []string{"memlock=unlimited", "RLIMIT_MEMLOCK=1024"},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rlimits, err := parseRlimits(tc.entries)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tc.expected, rlimits)
		})
	}
}

func TestRlimitModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description    string
		rlimits        []specs.POSIXRlimit
		spec           *specs.Spec
		expectedError  bool
		expectedLimits []specs.POSIXRlimit
	}{
		{
			description:   "nil spec is rejected",
			rlimits:       []specs.POSIXRlimit{{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64}},
			expectedError: true,
		},
		{
			description: "nil process is initialized",
			rlimits:     []specs.POSIXRlimit{{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64}},
			spec:        &specs.Spec{},
			expectedLimits: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
			},
		},
		{
			description: "existing rlimit of the same type is replaced",
			rlimits:     []specs.POSIXRlimit{{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64}},
			spec: &specs.Spec{
				Process: &specs.Process{
					Rlimits: []specs.POSIXRlimit{
						{Type: "RLIMIT_MEMLOCK", Soft: 65536, Hard: 65536},
						{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 1024},
					},
				},
			},
			expectedLimits: []specs.POSIXRlimit{
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
				{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 1024},
			},
		},
		{
			description: "rlimits of other types are appended",
			rlimits:     []specs.POSIXRlimit{{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64}},
			spec: &specs.Spec{
				Process: &specs.Process{
					Rlimits: []specs.POSIXRlimit{
						{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 1024},
					},
				},
			},
			expectedLimits: []specs.POSIXRlimit{
				{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 1024},
				{Type: "RLIMIT_MEMLOCK", Soft: math.MaxUint64, Hard: math.MaxUint64},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m := rlimitModifier{
				logger:  logger,
				rlimits: tc.rlimits,
			}
			err := m.Modify(tc.spec)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedLimits, tc.spec.Process.Rlimits)
		})
	}
}

func TestNewRlimitModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description      string
		rlimits          []string
		expectedError    bool
		expectedModifier bool
	}{
		{
			description: "no configured rlimits yield no modifier",
		},
		{
			description:      "configured rlimits yield a modifier",
			rlimits:          []string{"memlock=unlimited"},
			expectedModifier: true,
		},
		{
			description:   "invalid rlimits raise an error",
			rlimits:       []string{"memlock=lots"},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			f := createFactory(
				WithLogger(logger),
				WithConfig(&config.Config{
					NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
						Rlimits: tc.rlimits,
					},
				}),
			)
			m, err := f.newRlimitModifier()
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.expectedModifier {
				require.NotNil(t, m)
			} else {
				require.Nil(t, m)
			}
		})
	}
}
