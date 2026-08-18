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
*/

package docker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/docker"
)

func TestUpdateConfigDefaultRuntime(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		setAsDefault               bool
		runtimeName                string
		expectedDefaultRuntimeName any
	}{
		{},
		{
			setAsDefault:               false,
			expectedDefaultRuntimeName: nil,
		},
		{
			setAsDefault:               true,
			runtimeName:                "NAME",
			expectedDefaultRuntimeName: "NAME",
		},
		{
			setAsDefault:               true,
			runtimeName:                "nvidia",
			expectedDefaultRuntimeName: "nvidia",
		},
	}

	for i, tc := range testCases {
		o := &container.Options{
			RuntimeName:  tc.runtimeName,
			RuntimeDir:   runtimeDir,
			SetAsDefault: tc.setAsDefault,
		}

		config := docker.Config(map[string]any{})

		err := o.UpdateConfig(&config)
		require.NoError(t, err, "%d: %v", i, tc)

		defaultRuntimeName := config["default-runtime"]
		require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName, "%d: %v", i, tc)
	}
}

func TestUpdateConfig(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		config         docker.Config
		setAsDefault   bool
		runtimeName    string
		expectedConfig map[string]any
	}{
		{
			config:       map[string]any{},
			setAsDefault: false,
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
						"args": []string{},
					},
				},
			},
		},
		{
			config:       map[string]any{},
			setAsDefault: false,
			runtimeName:  "NAME",
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"NAME": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "nvidia-container-runtime",
						"args": []string{},
					},
				},
			},
			setAsDefault: false,
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
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
			expectedConfig: map[string]any{
				"runtimes": map[string]any{
					"not-nvidia": map[string]any{
						"path": "some-other-path",
						"args": []string{},
					},
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
						"args": []string{},
					},
				},
			},
		},
		{
			config: map[string]any{
				"default-runtime": "runc",
			},
			setAsDefault: true,
			runtimeName:  "nvidia",
			expectedConfig: map[string]any{
				"default-runtime": "nvidia",
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
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
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
						"args": []string{},
					},
				},
			},
		},
	}

	for i, tc := range testCases {

		o := &container.Options{
			RuntimeName:  tc.runtimeName,
			RuntimeDir:   runtimeDir,
			SetAsDefault: tc.setAsDefault,
		}

		err := o.UpdateConfig(&tc.config)
		require.NoError(t, err, "%d: %v", i, tc)

		configContent, err := json.MarshalIndent(tc.config, "", "    ")
		require.NoError(t, err)

		expectedContent, err := json.MarshalIndent(tc.expectedConfig, "", "    ")
		require.NoError(t, err)

		require.EqualValues(t, string(expectedContent), string(configContent), "%d: %v", i, tc)
	}
}

func TestRevertConfig(t *testing.T) {
	testCases := []struct {
		config         docker.Config
		expectedConfig map[string]any
	}{
		{
			config:         map[string]any{},
			expectedConfig: map[string]any{},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
				},
			},
			expectedConfig: map[string]any{},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
				},
			},
			expectedConfig: map[string]any{},
		},
		{
			config: map[string]any{
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
					"nvidia-cdi": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.cdi",
						"args": []string{},
					},
					"nvidia-legacy": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime.legacy",
						"args": []string{},
					},
				},
			},
			expectedConfig: map[string]any{},
		},
		{
			config: map[string]any{
				"default-runtime": "nvidia",
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
				},
			},
			expectedConfig: map[string]any{
				"default-runtime": "runc",
			},
		},
		{
			config: map[string]any{
				"default-runtime": "not-nvidia",
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
				},
			},
			expectedConfig: map[string]any{
				"default-runtime": "not-nvidia",
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
				"runtimes": map[string]any{
					"nvidia": map[string]any{
						"path": "/test/runtime/dir/nvidia-container-runtime",
						"args": []string{},
					},
				},
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
		o := &container.Options{}
		err := o.RevertConfig(&tc.config)

		require.NoError(t, err, "%d: %v", i, tc)

		configContent, err := json.MarshalIndent(tc.config, "", "    ")
		require.NoError(t, err)

		expectedContent, err := json.MarshalIndent(tc.expectedConfig, "", "    ")
		require.NoError(t, err)

		require.EqualValues(t, string(expectedContent), string(configContent), "%d: %v", i, tc)
	}
}
