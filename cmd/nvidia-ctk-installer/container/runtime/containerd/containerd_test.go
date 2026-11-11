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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
)

func TestConfigureWithoutDropInUpdatesTopLevelConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	initialConfig := `version = 2

[plugins."io.containerd.grpc.v1.cri".containerd]
  default_runtime_name = "runc"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"
`
	require.NoError(t, os.WriteFile(configPath, []byte(initialConfig), 0600))

	containerOptions := &container.Options{
		TopLevelConfigPath: configPath,
		ConfigSources:      []string{"file"},
		RuntimeName:        "nvidia",
		RuntimeDir:         "/usr/bin",
	}
	config, err := getRuntimeConfig(containerOptions, &Options{runtimeType: defaultRuntimeType})
	require.NoError(t, err)
	require.NoError(t, containerOptions.Configure(config))

	updated, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Contains(t, string(updated), `default_runtime_name = "runc"`)
	require.Contains(t, string(updated), "nvidia-container-runtime")
	require.NotContains(t, string(updated), "imports =")
}

func TestRuntimeOptions(t *testing.T) {
	testCases := []struct {
		description   string
		options       Options
		expected      map[string]any
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
			expected:      map[string]any{},
			expectedError: nil,
		},
		{
			description: "SystemdCgroup is true",
			options: Options{
				runtimeConfigOverrideJSON: "{\"SystemdCgroup\": true}",
			},
			expected: map[string]any{
				"SystemdCgroup": true,
			},
			expectedError: nil,
		},
		{
			description: "SystemdCgroup is false",
			options: Options{
				runtimeConfigOverrideJSON: "{\"SystemdCgroup\": false}",
			},
			expected: map[string]any{
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
