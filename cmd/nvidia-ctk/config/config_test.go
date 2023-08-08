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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestSetFlagToKeyValue(t *testing.T) {
	testCases := []struct {
		description   string
		config        map[string]interface{}
		setFlag       string
		expectedKey   string
		expectedValue interface{}
		expectedError error
	}{
		{
			description:   "empty config returns an error",
			setFlag:       "anykey=value",
			expectedKey:   "anykey",
			expectedError: errInvalidConfigOption,
		},
		{
			description: "option not present returns an error",
			config: map[string]interface{}{
				"defined": "defined-value",
			},
			setFlag:       "undefined=new-value",
			expectedKey:   "undefined",
			expectedError: errInvalidConfigOption,
		},
		{
			description: "boolean option assumes true",
			config: map[string]interface{}{
				"boolean": false,
			},
			setFlag:       "boolean",
			expectedKey:   "boolean",
			expectedValue: true,
		},
		{
			description: "boolean option returns true",
			config: map[string]interface{}{
				"boolean": false,
			},
			setFlag:       "boolean=true",
			expectedKey:   "boolean",
			expectedValue: true,
		},
		{
			description: "boolean option returns false",
			config: map[string]interface{}{
				"boolean": false,
			},
			setFlag:       "boolean=false",
			expectedKey:   "boolean",
			expectedValue: false,
		},
		{
			description: "invalid boolean option returns error",
			config: map[string]interface{}{
				"boolean": false,
			},
			setFlag:       "boolean=something",
			expectedKey:   "boolean",
			expectedValue: "something",
			expectedError: errInvalidFormat,
		},
		{
			description: "string option requires value",
			config: map[string]interface{}{
				"string": "value",
			},
			setFlag:       "string",
			expectedKey:   "string",
			expectedValue: nil,
			expectedError: errInvalidFormat,
		},
		{
			description: "string option returns value",
			config: map[string]interface{}{
				"string": "value",
			},
			setFlag:       "string=string-value",
			expectedKey:   "string",
			expectedValue: "string-value",
		},
		{
			description: "string option returns value with equals",
			config: map[string]interface{}{
				"string": "value",
			},
			setFlag:       "string=string-value=more",
			expectedKey:   "string",
			expectedValue: "string-value=more",
		},
		{
			description: "string option treats bool value as string",
			config: map[string]interface{}{
				"string": "value",
			},
			setFlag:       "string=true",
			expectedKey:   "string",
			expectedValue: "true",
		},
		{
			description: "string option treats int value as string",
			config: map[string]interface{}{
				"string": "value",
			},
			setFlag:       "string=5",
			expectedKey:   "string",
			expectedValue: "5",
		},
		{
			description: "[]string option returns single value",
			config: map[string]interface{}{
				"string": []string{"value"},
			},
			setFlag:       "string=string-value",
			expectedKey:   "string",
			expectedValue: []string{"string-value"},
		},
		{
			description: "[]string option returns multiple values",
			config: map[string]interface{}{
				"string": []string{"value"},
			},
			setFlag:       "string=first,second",
			expectedKey:   "string",
			expectedValue: []string{"first", "second"},
		},
		{
			description: "[]string option returns values with equals",
			config: map[string]interface{}{
				"string": []string{"value"},
			},
			setFlag:       "string=first=1,second=2",
			expectedKey:   "string",
			expectedValue: []string{"first=1", "second=2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tree, _ := toml.TreeFromMap(tc.config)
			cfgToml := (*config.Toml)(tree)
			k, v, err := (*configToml)(cfgToml).setFlagToKeyValue(tc.setFlag)
			require.ErrorIs(t, err, tc.expectedError)
			require.EqualValues(t, tc.expectedKey, k)
			require.EqualValues(t, tc.expectedValue, v)
		})
	}
}
