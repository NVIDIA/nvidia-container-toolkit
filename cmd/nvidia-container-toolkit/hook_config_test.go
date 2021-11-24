/**
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
**/

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHookConfig(t *testing.T) {
	testCases := []struct {
		lines                      []string
		expectedPanic              bool
		expectedDriverCapabilities DriverCapabilities
	}{
		{
			expectedDriverCapabilities: allDriverCapabilities,
		},
		{
			lines: []string{
				"supported-driver-capabilities = \"all\"",
			},
			expectedDriverCapabilities: allDriverCapabilities,
		},
		{
			lines: []string{
				"supported-driver-capabilities = \"compute,utility,not-compute\"",
			},
			expectedPanic: true,
		},
		{
			lines:                      []string{},
			expectedDriverCapabilities: allDriverCapabilities,
		},
		{
			lines: []string{
				"supported-driver-capabilities = \"\"",
			},
			expectedDriverCapabilities: none,
		},
		{
			lines: []string{
				"supported-driver-capabilities = \"utility,compute\"",
			},
			expectedDriverCapabilities: DriverCapabilities("utility,compute"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			var filename string
			defer func() {
				if len(filename) > 0 {
					os.Remove(filename)
				}
				configflag = nil
			}()

			if tc.lines != nil {
				configFile, err := os.CreateTemp("", "*.toml")
				require.NoError(t, err)
				defer configFile.Close()

				filename = configFile.Name()
				configflag = &filename

				for _, line := range tc.lines {
					_, err := configFile.WriteString(fmt.Sprintf("%s\n", line))
					require.NoError(t, err)
				}
			}

			var config HookConfig
			getHookConfig := func() {
				config = getHookConfig()
			}

			if tc.expectedPanic {
				require.Panics(t, getHookConfig)
				return
			}

			getHookConfig()

			require.EqualValues(t, tc.expectedDriverCapabilities, config.SupportedDriverCapabilities)
		})
	}
}
