/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetFlagToKeyValue(t *testing.T) {
	// TODO: We need to enable this test again since switching to reflect.
	testCases := []struct {
		description   string
		setFlag       string
		expectedKey   string
		expectedValue interface{}
		expectedError error
	}{
		{
			description:   "option not present returns an error",
			setFlag:       "undefined=new-value",
			expectedKey:   "undefined",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "undefined nexted option returns error",
			setFlag:       "nvidia-container-cli.undefined",
			expectedKey:   "nvidia-container-cli.undefined",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "boolean option assumes true",
			setFlag:       "disable-require",
			expectedKey:   "disable-require",
			expectedValue: true,
		},
		{
			description:   "boolean option returns true",
			setFlag:       "disable-require=true",
			expectedKey:   "disable-require",
			expectedValue: true,
		},
		{
			description:   "boolean option returns false",
			setFlag:       "disable-require=false",
			expectedKey:   "disable-require",
			expectedValue: false,
		},
		{
			description:   "invalid boolean option returns error",
			setFlag:       "disable-require=something",
			expectedKey:   "disable-require",
			expectedValue: "something",
			expectedError: errInvalidFormat,
		},
		{
			description:   "string option requires value",
			setFlag:       "swarm-resource",
			expectedKey:   "swarm-resource",
			expectedValue: nil,
			expectedError: errInvalidFormat,
		},
		{
			description:   "string option returns value",
			setFlag:       "swarm-resource=string-value",
			expectedKey:   "swarm-resource",
			expectedValue: "string-value",
		},
		{
			description:   "string option returns value with equals",
			setFlag:       "swarm-resource=string-value=more",
			expectedKey:   "swarm-resource",
			expectedValue: "string-value=more",
		},
		{
			description:   "string option treats bool value as string",
			setFlag:       "swarm-resource=true",
			expectedKey:   "swarm-resource",
			expectedValue: "true",
		},
		{
			description:   "string option treats int value as string",
			setFlag:       "swarm-resource=5",
			expectedKey:   "swarm-resource",
			expectedValue: "5",
		},
		{
			description:   "[]string option returns single value",
			setFlag:       "nvidia-container-cli.environment=string-value",
			expectedKey:   "nvidia-container-cli.environment",
			expectedValue: []string{"string-value"},
		},
		{
			description:   "[]string option returns multiple values",
			setFlag:       "nvidia-container-cli.environment=first,second",
			expectedKey:   "nvidia-container-cli.environment",
			expectedValue: []string{"first", "second"},
		},
		{
			description:   "[]string option returns values with equals",
			setFlag:       "nvidia-container-cli.environment=first=1,second=2",
			expectedKey:   "nvidia-container-cli.environment",
			expectedValue: []string{"first=1", "second=2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			k, v, err := setFlagToKeyValue(tc.setFlag)
			require.ErrorIs(t, err, tc.expectedError)
			require.EqualValues(t, tc.expectedKey, k)
			require.EqualValues(t, tc.expectedValue, v)
		})
	}
}
