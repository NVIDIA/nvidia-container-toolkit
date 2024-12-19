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
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = ""
					runtime_root = ""
					runtime_type = "io.containerd.runc.v2"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
			`,
			expectedError: nil,
		},
		{
			description: "options from runc are imported",
			config: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from default runtime are imported",
			config: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				default_runtime_name = "default"
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				default_runtime_name = "default"
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from the default runtime take precedence over runc",
			config: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				default_runtime_name = "default"
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
			`,
			expectedConfig: `
			version = 2
			[plugins]
			[plugins."io.containerd.grpc.v1.cri"]
				[plugins."io.containerd.grpc.v1.cri".containerd]
				default_runtime_name = "default"
				[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						SystemdCgroup = false
				`,
		},
		{
			description: "empty v3 spec is supported",
			config: `
			version = 3
			`,
			expectedConfig: `
			version = 3
			[plugins]
			[plugins."io.containerd.cri.v1.runtime"]
				[plugins."io.containerd.cri.v1.runtime".containerd]
				[plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = ""
					runtime_root = ""
					runtime_type = "io.containerd.runc.v2"
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
			`,
			expectedError: nil,
		},
		{
			description: "v3 spec is supported",
			config: `
			version = 3
			[plugins]
			[plugins."io.containerd.cri.v1.runtime"]
				[plugins."io.containerd.cri.v1.runtime".containerd]
				[plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 3
			[plugins]
			[plugins."io.containerd.cri.v1.runtime"]
				[plugins."io.containerd.cri.v1.runtime".containerd]
				[plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						SystemdCgroup = true
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
			)
			require.NoError(t, err)

			err = c.AddRuntime("test", "/usr/bin/test", tc.setAsDefault)
			require.NoError(t, err)

			require.EqualValues(t, expectedConfig.String(), c.String())
		})
	}
}

func TestGetRuntimeConfig(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	config := `
	version = 2
	[plugins]
	[plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      ignore_blockio_not_enabled_errors = false
      ignore_rdt_not_enabled_errors = false
      no_pivot = false
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        privileged_without_host_devices_all_devices_allowed = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""
        sandbox_mode = ""
        snapshotter = ""

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          base_runtime_spec = ""
          cni_conf_dir = ""
          cni_max_conf_num = 0
          container_annotations = []
          pod_annotations = []
          privileged_without_host_devices = false
          privileged_without_host_devices_all_devices_allowed = false
          runtime_engine = ""
          runtime_path = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"
          sandbox_mode = "podsandbox"
          snapshotter = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            CriuImagePath = ""
            CriuPath = ""
            CriuWorkPath = ""
            IoGid = 0
            IoUid = 0
            NoNewKeyring = false
            NoPivotRoot = false
            Root = ""
            ShimCgroup = ""
            SystemdCgroup = false
`
	testCases := []struct {
		description   string
		runtime       string
		expected      string
		expectedError error
	}{
		{
			description:   "valid runtime config, existing runtime",
			runtime:       "runc",
			expected:      "/usr/bin/runc",
			expectedError: nil,
		},
		{
			description:   "valid runtime config, non-existing runtime",
			runtime:       "some-other-runtime",
			expected:      "",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(config)),
			)
			require.NoError(t, err)

			rc, err := c.GetRuntimeConfig(tc.runtime)
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expected, rc.GetBinaryPath())
		})
	}
}
