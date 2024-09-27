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

package crio

import (
	"testing"

	"github.com/pelletier/go-toml"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestAddRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description    string
		config         string
		setAsDefault   bool
		expectedConfig string
		expectedError  error
	}{
		{
			description: "empty config not default runtime",
			expectedConfig: `
			[crio]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedError: nil,
		},
		{
			description: "options from runc are imported",
			config: `
			[crio]
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			runc_option = "option"
			`,
		},
		{
			description: "options from default runtime are imported",
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			default_option = "option"
			`,
		},
		{
			description: "options from runc take precedence over default runtime",
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			runc_option = "option"
			`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			config, err := toml.Load(tc.config)
			require.NoError(t, err)
			expectedConfig, err := toml.Load(tc.expectedConfig)
			require.NoError(t, err)

			c := &Config{
				Logger: logger,
				Tree:   config,
			}

			err = c.AddRuntime("test", "/usr/bin/test", tc.setAsDefault)
			require.NoError(t, err)

			require.EqualValues(t, expectedConfig.String(), config.String())
		})
	}
}
