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

package containerd

import (
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

func TestUpdateV1ConfigDefaultRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		legacyConfig                 bool
		setAsDefault                 bool
		runtimeName                  string
		expectedDefaultRuntimeName   interface{}
		expectedDefaultRuntimeBinary interface{}
	}{
		{},
		{
			legacyConfig:                 true,
			setAsDefault:                 false,
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: nil,
		},
		{
			legacyConfig:                 true,
			setAsDefault:                 true,
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: "/test/runtime/dir/nvidia-container-runtime",
		},
		{
			legacyConfig:                 true,
			setAsDefault:                 true,
			runtimeName:                  "NAME",
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: "/test/runtime/dir/nvidia-container-runtime",
		},
		{
			legacyConfig:                 false,
			setAsDefault:                 false,
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: nil,
		},
		{
			legacyConfig:                 false,
			setAsDefault:                 true,
			expectedDefaultRuntimeName:   "nvidia",
			expectedDefaultRuntimeBinary: nil,
		},
		{
			legacyConfig:                 false,
			setAsDefault:                 true,
			runtimeName:                  "NAME",
			expectedDefaultRuntimeName:   "NAME",
			expectedDefaultRuntimeBinary: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &container.Options{
				RuntimeName:  tc.runtimeName,
				RuntimeDir:   runtimeDir,
				SetAsDefault: tc.setAsDefault,
			}

			v1, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.Empty),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithUseLegacyConfig(tc.legacyConfig),
				containerd.WithConfigVersion(1),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v1)
			require.NoError(t, err)

			cfg := v1.(*containerd.ConfigV1)
			defaultRuntimeName := cfg.GetPath([]string{"plugins", "cri", "containerd", "default_runtime_name"})
			require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName)

			defaultRuntime := cfg.GetPath([]string{"plugins", "cri", "containerd", "default_runtime"})
			if tc.expectedDefaultRuntimeBinary == nil {
				require.Nil(t, defaultRuntime)
			} else {
				require.NotNil(t, defaultRuntime)

				expected, err := defaultRuntimeTomlConfigV1(tc.expectedDefaultRuntimeBinary.(string))
				require.NoError(t, err)

				configContents, _ := toml.Marshal(defaultRuntime.(*toml.Tree))
				expectedContents, _ := toml.Marshal(expected)

				require.Equal(t, string(expectedContents), string(configContents), "%d: %v: %v", i, tc)
			}
		})
	}
}

func TestUpdateV1Config(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		runtimeName    string
		expectedConfig map[string]interface{}
	}{
		{
			runtimeName: "nvidia",
			expectedConfig: map[string]interface{}{
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime",
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime.cdi",
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime.legacy",
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
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime",
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime.cdi",
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
										"Runtime":    "/test/runtime/dir/nvidia-container-runtime.legacy",
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
			o := &container.Options{
				RuntimeName: tc.runtimeName,
				RuntimeDir:  runtimeDir,
			}

			v1, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.Empty),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithConfigVersion(1),
				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v1)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)

			require.Equal(t, expected.String(), v1.String())
		})
	}
}

func TestUpdateV1ConfigWithRuncPresent(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		runtimeName    string
		expectedConfig map[string]interface{}
	}{
		{
			runtimeName: "nvidia",
			expectedConfig: map[string]interface{}{
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime",
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime.cdi",
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime.legacy",
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
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime",
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime.cdi",
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
										"Runtime":     "/test/runtime/dir/nvidia-container-runtime.legacy",
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
			o := &container.Options{
				RuntimeName: tc.runtimeName,
				RuntimeDir:  runtimeDir,
			}

			v1, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.FromMap(runcConfigMapV1("/runc-binary"))),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithConfigVersion(1),
				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v1)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)

			require.Equal(t, expected.String(), v1.String())
		})
	}
}

func TestUpdateV1EnableCDI(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		enableCDI              bool
		expectedEnableCDIValue interface{}
	}{
		{},
		{
			enableCDI:              false,
			expectedEnableCDIValue: nil,
		},
		{
			enableCDI:              true,
			expectedEnableCDIValue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.enableCDI), func(t *testing.T) {
			o := &container.Options{
				EnableCDI:   tc.enableCDI,
				RuntimeName: "nvidia",
				RuntimeDir:  runtimeDir,
			}

			cfg, err := toml.Empty.Load()
			require.NoError(t, err)

			v1 := &containerd.ConfigV1{
				Logger:      logger,
				Tree:        cfg,
				RuntimeType: runtimeType,
			}

			err = o.UpdateConfig(v1)
			require.NoError(t, err)

			enableCDIValue := v1.GetPath([]string{"plugins", "cri", "containerd", "enable_cdi"})
			require.EqualValues(t, tc.expectedEnableCDIValue, enableCDIValue)
		})
	}
}

func TestRevertV1Config(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		config map[string]interface {
		}
		expected map[string]interface{}
	}{
		{},
		{
			config: map[string]interface{}{
				"version": int64(1),
			},
		},
		{
			config: map[string]interface{}{
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"nvidia":        runtimeMapV1("/test/runtime/dir/nvidia-container-runtime"),
								"nvidia-cdi":    runtimeMapV1("/test/runtime/dir/nvidia-container-runtime.cdi"),
								"nvidia-legacy": runtimeMapV1("/test/runtime/dir/nvidia-container-runtime.legacy"),
							},
						},
					},
				},
			},
		},
		{
			config: map[string]interface{}{
				"version": int64(1),
				"plugins": map[string]interface{}{
					"cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"nvidia":        runtimeMapV1("/test/runtime/dir/nvidia-container-runtime"),
								"nvidia-cdi":    runtimeMapV1("/test/runtime/dir/nvidia-container-runtime.cdi"),
								"nvidia-legacy": runtimeMapV1("/test/runtime/dir/nvidia-container-runtime.legacy"),
							},
							"default_runtime":      defaultRuntimeV1("/test/runtime/dir/nvidia-container-runtime"),
							"default_runtime_name": "nvidia",
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o := &container.Options{
				RuntimeName: "nvidia",
			}

			expected, err := toml.TreeFromMap(tc.expected)
			require.NoError(t, err)

			v1, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.FromMap(tc.config)),
				containerd.WithRuntimeType(runtimeType),

				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.RevertConfig(v1)
			require.NoError(t, err)

			require.Equal(t, expected.String(), v1.String())
		})
	}
}

func defaultRuntimeTomlConfigV1(binary string) (*toml.Tree, error) {
	return toml.TreeFromMap(defaultRuntimeV1(binary))
}

func defaultRuntimeV1(binary string) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type":                    runtimeType,
		"runtime_root":                    "",
		"runtime_engine":                  "",
		"privileged_without_host_devices": false,
		"options": map[string]interface{}{
			"BinaryName": binary,
			"Runtime":    binary,
		},
	}
}

func runtimeMapV1(binary string) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type":                    runtimeType,
		"runtime_root":                    "",
		"runtime_engine":                  "",
		"privileged_without_host_devices": false,
		"options": map[string]interface{}{
			"BinaryName": binary,
			"Runtime":    binary,
		},
	}
}

func runcConfigMapV1(binary string) map[string]interface{} {
	return map[string]interface{}{
		"plugins": map[string]interface{}{
			"cri": map[string]interface{}{
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
