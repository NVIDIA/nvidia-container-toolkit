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

package main

import (
	"fmt"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/tools/container"
)

const (
	runtimeType = "runtime_type"
)

func TestUpdateV2ConfigDefaultRuntime(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		setAsDefault               bool
		runtimeName                string
		expectedDefaultRuntimeName interface{}
	}{
		{},
		{
			setAsDefault:               false,
			runtimeName:                "nvidia",
			expectedDefaultRuntimeName: nil,
		},
		{
			setAsDefault:               false,
			runtimeName:                "NAME",
			expectedDefaultRuntimeName: nil,
		},
		{
			setAsDefault:               true,
			runtimeName:                "nvidia",
			expectedDefaultRuntimeName: "nvidia",
		},
		{
			setAsDefault:               true,
			runtimeName:                "NAME",
			expectedDefaultRuntimeName: "NAME",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &options{
				Options: container.Options{
					RuntimeName:  tc.runtimeName,
					RuntimeDir:   runtimeDir,
					SetAsDefault: tc.setAsDefault,
				},
			}

			config, err := toml.TreeFromMap(map[string]interface{}{})
			require.NoError(t, err)

			v2 := &containerd.Config{
				Tree:        config,
				RuntimeType: runtimeType,
			}

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			defaultRuntimeName := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
			require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName)
		})
	}
}

func TestUpdateV2Config(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		runtimeName    string
		expectedConfig map[string]interface{}
	}{
		{
			runtimeName: "nvidia",
			expectedConfig: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"nvidia": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime",
									},
								},
								"nvidia-cdi": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.cdi",
									},
								},
								"nvidia-legacy": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.legacy",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			runtimeName: "NAME",
			expectedConfig: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"NAME": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime",
									},
								},
								"nvidia-cdi": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.cdi",
									},
								},
								"nvidia-legacy": map[string]interface{}{
									"runtime_type":                    "runtime_type",
									"runtime_root":                    "",
									"runtime_engine":                  "",
									"privileged_without_host_devices": false,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.legacy",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &options{
				Options: container.Options{
					RuntimeName: tc.runtimeName,
					RuntimeDir:  runtimeDir,
				},
				runtimeType: runtimeType,
			}

			config, err := toml.TreeFromMap(map[string]interface{}{})
			require.NoError(t, err)

			v2 := &containerd.Config{
				Tree:                 config,
				RuntimeType:          runtimeType,
				ContainerAnnotations: []string{"cdi.k8s.io/*"},
			}

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)

			require.Equal(t, expected.String(), config.String())
		})
	}

}

func TestUpdateV2ConfigWithRuncPresent(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		runtimeName    string
		expectedConfig map[string]interface{}
	}{
		{
			runtimeName: "nvidia",
			expectedConfig: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"runc": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/runc-binary",
									},
								},
								"nvidia": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime",
									},
								},
								"nvidia-cdi": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.cdi",
									},
								},
								"nvidia-legacy": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.legacy",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			runtimeName: "NAME",
			expectedConfig: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"runc": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/runc-binary",
									},
								},
								"NAME": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime",
									},
								},
								"nvidia-cdi": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.cdi",
									},
								},
								"nvidia-legacy": map[string]interface{}{
									"runtime_type":                    "runc_runtime_type",
									"runtime_root":                    "runc_runtime_root",
									"runtime_engine":                  "runc_runtime_engine",
									"privileged_without_host_devices": true,
									"container_annotations":           []string{"cdi.k8s.io/*"},
									"options": map[string]interface{}{
										"runc-option": "value",
										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.legacy",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &options{
				Options: container.Options{
					RuntimeName: tc.runtimeName,
					RuntimeDir:  runtimeDir,
				},
			}

			config, err := toml.TreeFromMap(runcConfigMapV2("/runc-binary"))
			require.NoError(t, err)

			v2 := &containerd.Config{
				Tree:                 config,
				RuntimeType:          runtimeType,
				ContainerAnnotations: []string{"cdi.k8s.io/*"},
			}

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)

			require.Equal(t, expected.String(), config.String())
		})
	}
}

func TestRevertV2Config(t *testing.T) {
	testCases := []struct {
		config map[string]interface {
		}
		expected map[string]interface{}
	}{
		{},
		{
			config: map[string]interface{}{
				"version": int64(2),
			},
		},
		{
			config: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"nvidia": runtimeMapV2("/test/runtime/dir/nvidia-container-runtime"),
							},
						},
					},
				},
			},
		},
		{
			config: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"nvidia": runtimeMapV2("/test/runtime/dir/nvidia-container-runtime"),
							},
							"default_runtime_name": "nvidia",
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &options{
				Options: container.Options{
					RuntimeName: "nvidia",
				},
			}

			config, err := toml.TreeFromMap(tc.config)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expected)
			require.NoError(t, err)

			v2 := &containerd.Config{
				Tree:        config,
				RuntimeType: runtimeType,
			}

			err = o.RevertConfig(v2)
			require.NoError(t, err)

			configContents, _ := toml.Marshal(config)
			expectedContents, _ := toml.Marshal(expected)

			require.Equal(t, string(expectedContents), string(configContents))
		})
	}
}

func runtimeMapV2(binary string) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type":                    runtimeType,
		"runtime_root":                    "",
		"runtime_engine":                  "",
		"privileged_without_host_devices": false,
		"options": map[string]interface{}{
			"BinaryName": binary,
		},
	}
}

func runcConfigMapV2(binary string) map[string]interface{} {
	return map[string]interface{}{
		"plugins": map[string]interface{}{
			"io.containerd.grpc.v1.cri": map[string]interface{}{
				"containerd": map[string]interface{}{
					"runtimes": map[string]interface{}{
						"runc": map[string]interface{}{
							"runtime_type":                    "runc_runtime_type",
							"runtime_root":                    "runc_runtime_root",
							"runtime_engine":                  "runc_runtime_engine",
							"privileged_without_host_devices": true,
							"options": map[string]interface{}{
								"runc-option": "value",
								"BinaryName":  binary,
							},
						},
					},
				},
			},
		},
	}
}
