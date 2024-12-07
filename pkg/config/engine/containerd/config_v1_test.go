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

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

func TestAddRuntimeV1(t *testing.T) {
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
			version = 1
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = ""
					runtime_root = ""
					runtime_type = ""
					[plugins.cri.containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
			`,
			expectedError: nil,
		},
		{
			description: "options from runc are imported",
			config: `
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins.cri.containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from default runtime are imported",
			config: `
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				default_runtime_name = "default"
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				default_runtime_name = "default"
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
					[plugins.cri.containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from the default runtime take precedence over runc",
			config: `
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				default_runtime_name = "default"
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins.cri.containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins.cri.containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins.cri]
				[plugins.cri.containerd]
				default_runtime_name = "default"
				[plugins.cri.containerd.runtimes]
					[plugins.cri.containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins.cri.containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins.cri.containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins.cri.containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
					[plugins.cri.containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins.cri.containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = false
				`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			expectedConfig, err := toml.Load(tc.expectedConfig)
			require.NoError(t, err)

			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(tc.config)),
				WithUseLegacyConfig(true),
				WithRuntimeType(""),
			)
			require.NoError(t, err)

			err = c.AddRuntime("test", "/usr/bin/test", tc.setAsDefault)
			require.NoError(t, err)

			require.EqualValues(t, expectedConfig.String(), c.String())
		})
	}
}
