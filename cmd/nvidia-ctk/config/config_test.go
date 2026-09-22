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
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestSetFlagToKeyValue(t *testing.T) {
	testCases := []struct {
		description      string
		setFlag          string
		setListSeparator string
		expectedKey      string
		expectedValue    any
		expectedError    error
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
			description:   "child of boolean option returns error",
			setFlag:       "disable-require.extra=true",
			expectedKey:   "disable-require.extra",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "child of string option returns error",
			setFlag:       "nvidia-container-cli.path.extra=/tmp/cli",
			expectedKey:   "nvidia-container-cli.path.extra",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "child of slice option returns error",
			setFlag:       "nvidia-container-cli.environment.extra=VALUE",
			expectedKey:   "nvidia-container-cli.environment.extra",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "child of pointer option returns error",
			setFlag:       "features.allow-ldconfig-from-container.extra=true",
			expectedKey:   "features.allow-ldconfig-from-container.extra",
			expectedError: errInvalidConfigOption,
		},
		{
			description:   "nested string option returns value",
			setFlag:       "nvidia-container-cli.path=/tmp/cli",
			expectedKey:   "nvidia-container-cli.path",
			expectedValue: "/tmp/cli",
		},
		{
			description:   "deeply nested string option returns value",
			setFlag:       "nvidia-container-runtime.modes.cdi.default-kind=example.com/gpu",
			expectedKey:   "nvidia-container-runtime.modes.cdi.default-kind",
			expectedValue: "example.com/gpu",
		},
		{
			description:   "pointer boolean option assumes true",
			setFlag:       "features.allow-ldconfig-from-container",
			expectedKey:   "features.allow-ldconfig-from-container",
			expectedValue: true,
		},
		{
			description:   "pointer boolean option returns true",
			setFlag:       "features.allow-ldconfig-from-container=true",
			expectedKey:   "features.allow-ldconfig-from-container",
			expectedValue: true,
		},
		{
			description:   "pointer boolean option returns false",
			setFlag:       "features.allow-ldconfig-from-container=false",
			expectedKey:   "features.allow-ldconfig-from-container",
			expectedValue: false,
		},
		{
			description: "pointer boolean option returns nil",
			setFlag:     "features.allow-ldconfig-from-container=nil",
			expectedKey: "features.allow-ldconfig-from-container",
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
			description:      "[]string option returns multiple values",
			setFlag:          "nvidia-container-cli.environment=first,second",
			setListSeparator: ",",
			expectedKey:      "nvidia-container-cli.environment",
			expectedValue:    []string{"first", "second"},
		},
		{
			description:      "[]string option returns values with equals",
			setFlag:          "nvidia-container-cli.environment=first=1,second=2",
			setListSeparator: ",",
			expectedKey:      "nvidia-container-cli.environment",
			expectedValue:    []string{"first=1", "second=2"},
		},
		{
			description:      "[]string option returns multiple values semi-colon",
			setFlag:          "nvidia-container-cli.environment=first;second",
			setListSeparator: ";",
			expectedKey:      "nvidia-container-cli.environment",
			expectedValue:    []string{"first", "second"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.setListSeparator == "" {
				tc.setListSeparator = ","
			}
			k, v, err := setFlagToKeyValue(tc.setFlag, tc.setListSeparator)
			require.ErrorIs(t, err, tc.expectedError)
			require.EqualValues(t, tc.expectedKey, k)
			require.EqualValues(t, tc.expectedValue, v)
		})
	}
}

func TestConfigCommandSet(t *testing.T) {
	testCases := map[string]struct {
		setFlag       string
		expectedKey   string
		expectedValue any
		expectedError error
	}{
		"child of boolean": {
			setFlag:       "disable-require.extra=true",
			expectedError: errInvalidConfigOption,
		},
		"child of string": {
			setFlag:       "nvidia-container-cli.path.extra=/tmp/cli",
			expectedError: errInvalidConfigOption,
		},
		"child of slice": {
			setFlag:       "nvidia-container-cli.environment.extra=VALUE",
			expectedError: errInvalidConfigOption,
		},
		"child of pointer": {
			setFlag:       "features.allow-ldconfig-from-container.extra=true",
			expectedError: errInvalidConfigOption,
		},
		"nested string": {
			setFlag:       "nvidia-container-cli.path=/tmp/cli",
			expectedKey:   "nvidia-container-cli.path",
			expectedValue: "/tmp/cli",
		},
		"deeply nested string": {
			setFlag:       "nvidia-container-runtime.modes.cdi.default-kind=example.com/gpu",
			expectedKey:   "nvidia-container-runtime.modes.cdi.default-kind",
			expectedValue: "example.com/gpu",
		},
		"slice": {
			setFlag:       "nvidia-container-cli.environment=FIRST=1:SECOND=2",
			expectedKey:   "nvidia-container-cli.environment",
			expectedValue: []any{"FIRST=1", "SECOND=2"},
		},
		"pointer boolean assumes true": {
			setFlag:       "features.allow-ldconfig-from-container",
			expectedKey:   "features.allow-ldconfig-from-container",
			expectedValue: true,
		},
		"pointer boolean false": {
			setFlag:       "features.allow-ldconfig-from-container=false",
			expectedKey:   "features.allow-ldconfig-from-container",
			expectedValue: false,
		},
		"pointer boolean nil": {
			setFlag:     "features.allow-ldconfig-from-container=nil",
			expectedKey: "features.allow-ldconfig-from-container",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.toml")
			original := []byte("disable-require = false\n[features]\nallow-ldconfig-from-container = true\n")
			require.NoError(t, os.WriteFile(configPath, original, 0600))

			var err error
			require.NotPanics(t, func() {
				cmd := NewCommand(nil)
				err = cmd.Run(t.Context(), []string{"config", "--config-file", configPath, "--in-place", "--set", tc.setFlag})
			})
			require.ErrorIs(t, err, tc.expectedError)

			contents, readErr := os.ReadFile(configPath)
			require.NoError(t, readErr)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, errUndefinedField)
				require.Contains(t, err.Error(), "invalid --set option "+tc.setFlag)
				require.Equal(t, original, contents)
				return
			}

			cfg, err := toml.LoadBytes(contents)
			require.NoError(t, err)
			require.Equal(t, tc.expectedValue, cfg.Get(tc.expectedKey))
		})
	}
}
