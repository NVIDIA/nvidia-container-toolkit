/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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
*/

package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateConfigDefaultRuntime(t *testing.T) {
	testCases := []struct {
		config                     Config
		runtimeName                string
		setAsDefault               bool
		expectedDefaultRuntimeName any
	}{
		{
			setAsDefault:               false,
			expectedDefaultRuntimeName: nil,
		},
		{
			runtimeName:                "NAME",
			setAsDefault:               true,
			expectedDefaultRuntimeName: "NAME",
		},
		{
			config: map[string]any{
				"default-runtime": "ALREADY_SET",
			},
			runtimeName:                "NAME",
			setAsDefault:               false,
			expectedDefaultRuntimeName: "ALREADY_SET",
		},
		{
			config: map[string]any{
				"default-runtime": "ALREADY_SET",
			},
			runtimeName:                "NAME",
			setAsDefault:               true,
			expectedDefaultRuntimeName: "NAME",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			if tc.config == nil {
				tc.config = make(map[string]any)
			}
			err := tc.config.AddRuntime(tc.runtimeName, "", tc.setAsDefault)
			require.NoError(t, err)

			defaultRuntimeName := tc.config["default-runtime"]
			require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName)
		})
	}
}

func TestUpdateConfigRuntimes(t *testing.T) {
	testCases := []struct {
		config         Config
		runtimes       map[string]string
		expectedConfig map[string]any
	}{
		{
			config: map[string]any{},
			runtimes: map[string]string{
				"runtime1": "/test/runtime/dir/runtime1",
				"runtime2": "/test/runtime/dir/runtime2",
			},
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"runtime1": map[string]any{
						"path": "/test/runtime/dir/runtime1",
						"args": []string{},
					},
					"runtime2": map[string]any{
						"path": "/test/runtime/dir/runtime2",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"runtime1": map[string]any{
						"path": "runtime1",
						"args": []string{},
					},
				},
			},
			runtimes: map[string]string{
				"runtime1": "/test/runtime/dir/runtime1",
				"runtime2": "/test/runtime/dir/runtime2",
			},
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"runtime1": map[string]any{
						"path": "/test/runtime/dir/runtime1",
						"args": []string{},
					},
					"runtime2": map[string]any{
						"path": "/test/runtime/dir/runtime2",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"not-nvidia": map[string]any{
						"path": "some-other-path",
						"args": []string{},
					},
				},
			},
			runtimes: map[string]string{
				"runtime1": "/test/runtime/dir/runtime1",
			},
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"not-nvidia": map[string]any{
						"path": "some-other-path",
						"args": []string{},
					},
					"runtime1": map[string]any{
						"path": "/test/runtime/dir/runtime1",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"exec-opts":  []string{"native.cgroupdriver=systemd"},
				"log-driver": "json-file",
				"log-opts": map[string]string{
					"max-size": "100m",
				},
				"storage-driver": "overlay2",
			},
			runtimes: map[string]string{
				"runtime1": "/test/runtime/dir/runtime1",
			},
			expectedConfig: map[string]any{
				"exec-opts":  []string{"native.cgroupdriver=systemd"},
				"log-driver": "json-file",
				"log-opts": map[string]string{
					"max-size": "100m",
				},
				"storage-driver": "overlay2",
				"runtimes": map[string]any{
					"runtime1": map[string]any{
						"path": "/test/runtime/dir/runtime1",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"exec-opts":  []string{"native.cgroupdriver=systemd"},
				"log-driver": "json-file",
				"log-opts": map[string]string{
					"max-size": "100m",
				},
				"storage-driver": "overlay2",
			},
			expectedConfig: map[string]any{
				"exec-opts":  []string{"native.cgroupdriver=systemd"},
				"log-driver": "json-file",
				"log-opts": map[string]string{
					"max-size": "100m",
				},
				"storage-driver": "overlay2",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			for runtimeName, runtimePath := range tc.runtimes {
				err := tc.config.AddRuntime(runtimeName, runtimePath, false)
				require.NoError(t, err)
			}

			configContent, err := json.MarshalIndent(tc.config, "", "    ")
			require.NoError(t, err)

			expectedContent, err := json.MarshalIndent(tc.expectedConfig, "", "    ")
			require.NoError(t, err)

			require.EqualValues(t, string(expectedContent), string(configContent))
		})

	}
}

func TestGetRuntimeConfig(t *testing.T) {
	c := map[string]any{
		"runtimes": map[string]any{
			"nvidia": map[string]any{
				"path": "nvidia-container-runtime",
				"args": []string{},
			},
		},
	}
	cfg := Config(c)

	testCases := []struct {
		description string
		runtime     string
		expected    string
	}{
		{
			description: "existing runtime",
			runtime:     "nvidia",
			expected:    "nvidia-container-runtime",
		},
		{
			description: "non-existent runtime",
			runtime:     "some-other-runtime",
			expected:    "",
		},
	}
	for _, tc := range testCases {
		rc, err := cfg.GetRuntimeConfig(tc.runtime)
		require.NoError(t, err)
		require.Equal(t, tc.expected, rc.GetBinaryPath())
	}
}

func TestEnableCDIPreservesFeaturesFromFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"missing features", `{}`, `{"features":{"cdi":true}}`},
		{"empty features", `{"features":{}}`, `{"features":{"cdi":true}}`},
		{"existing flags", `{"features":{"containerd-snapshotter":true,"buildkit":false},"debug":true}`, `{"features":{"cdi":true,"containerd-snapshotter":true,"buildkit":false},"debug":true}`},
		{"CDI disabled", `{"features":{"cdi":false,"containerd-snapshotter":true}}`, `{"features":{"cdi":true,"containerd-snapshotter":true}}`},
		{"CDI enabled", `{"features":{"cdi":true,"containerd-snapshotter":true}}`, `{"features":{"cdi":true,"containerd-snapshotter":true}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "daemon.json")
			require.NoError(t, os.WriteFile(path, []byte(tc.input), 0o600))
			for i := 0; i < 2; i++ {
				cfg, err := New(WithPath(path))
				require.NoError(t, err)
				cfg.EnableCDI()
				_, err = cfg.Save(path)
				require.NoError(t, err)
				contents, err := os.ReadFile(path)
				require.NoError(t, err)
				require.JSONEq(t, tc.expected, string(contents))
			}
		})
	}
}
