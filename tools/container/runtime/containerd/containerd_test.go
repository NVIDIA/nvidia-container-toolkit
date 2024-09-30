/**
# Copyright 2024 NVIDIA CORPORATION
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

package containerd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeOptions(t *testing.T) {
	testCases := []struct {
		description   string
		options       Options
		expected      map[string]interface{}
		expectedError error
	}{
		{
			description: "empty is nil",
		},
		{
			description: "empty json",
			options: Options{
				runtimeConfigOverrideJSON: "{}",
			},
			expected:      map[string]interface{}{},
			expectedError: nil,
		},
		{
			description: "SystemdCgroup is true",
			options: Options{
				runtimeConfigOverrideJSON: "{\"SystemdCgroup\": true}",
			},
			expected: map[string]interface{}{
				"SystemdCgroup": true,
			},
			expectedError: nil,
		},
		{
			description: "SystemdCgroup is false",
			options: Options{
				runtimeConfigOverrideJSON: "{\"SystemdCgroup\": false}",
			},
			expected: map[string]interface{}{
				"SystemdCgroup": false,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			runtimeOptions, err := tc.options.runtimeConfigOverride()
			require.ErrorIs(t, tc.expectedError, err)
			require.EqualValues(t, tc.expected, runtimeOptions)
		})
	}
}
