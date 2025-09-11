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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

const (
	runtimeType = "runtime_type"
)

func TestSetupCleanup(t *testing.T) {
	c := &cli.Command{
		Name: "test",
	}
	testCases := []struct {
		description                 string
		containerOptions            container.Options
		options                     Options
		prepareEnvironment          func(*testing.T, string) error
		expectedSetupError          error
		assertSetupPostConditions   func(*testing.T, string) error
		expectedCleanupError        error
		assertCleanupPostConditions func(*testing.T, string) error
	}{
		{
			description: "top-level config does not exist",
			containerOptions: container.Options{
				DropInConfig: "{{ .testRoot }}/conf.d/99-nvidia.toml",
				Config:       "{{ .testRoot }}/etc/containerd/config.toml",
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				require.FileExists(t, filepath.Join(testRoot, "/etc/containerd/config.toml"))
				topLevel, err := toml.FromFile(filepath.Join(testRoot, "/etc/containerd/config.toml")).Load()
				require.NoError(t, err)
				require.EqualValues(t, `imports = ["`+filepath.Join(testRoot, "/conf.d/*.toml")+`"]
version = 2
`, topLevel.String())
				require.FileExists(t, filepath.Join(testRoot, "/conf.d/99-nvidia.toml"))
				dropIn, err := toml.FromFile(filepath.Join(testRoot, "/conf.d/99-nvidia.toml")).Load()
				require.NoError(t, err)
				require.EqualValues(t, `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "nvidia-container-runtime.legacy"
`, dropIn.String())
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				require.NoFileExists(t, filepath.Join(testRoot, "/etc/containerd/config.toml"))
				require.NoFileExists(t, filepath.Join(testRoot, "/conf.d/99-nvidia.toml"))
				return nil
			},
		},
	}

	for _, tc := range testCases {
		// TODO: Add tests for restart mode.
		tc.containerOptions.RestartMode = "none"
		t.Run(tc.description, func(t *testing.T) {
			// Update any paths as required:
			testRoot := t.TempDir()
			tc.containerOptions.Config = strings.ReplaceAll(tc.containerOptions.Config, "{{ .testRoot }}", testRoot)
			tc.containerOptions.DropInConfig = strings.ReplaceAll(tc.containerOptions.DropInConfig, "{{ .testRoot }}", testRoot)

			// Prepare the test environment
			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, testRoot))
			}

			err := Setup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedSetupError, err)

			if tc.assertSetupPostConditions != nil {
				require.NoError(t, tc.assertSetupPostConditions(t, testRoot))
			}

			err = Cleanup(c, &tc.containerOptions, &tc.options)
			require.EqualValues(t, tc.expectedCleanupError, err)

			if tc.assertCleanupPostConditions != nil {
				require.NoError(t, tc.assertCleanupPostConditions(t, testRoot))
			}
		})
	}
}

// func TestUpdateV2ConfigDefaultRuntime(t *testing.T) {
// 	logger, _ := testlog.NewNullLogger()
// 	const runtimeDir = "/test/runtime/dir"

// 	testCases := []struct {
// 		setAsDefault               bool
// 		runtimeName                string
// 		expectedDefaultRuntimeName interface{}
// 	}{
// 		{},
// 		{
// 			setAsDefault:               false,
// 			runtimeName:                "nvidia",
// 			expectedDefaultRuntimeName: nil,
// 		},
// 		{
// 			setAsDefault:               false,
// 			runtimeName:                "NAME",
// 			expectedDefaultRuntimeName: nil,
// 		},
// 		{
// 			setAsDefault:               true,
// 			runtimeName:                "nvidia",
// 			expectedDefaultRuntimeName: "nvidia",
// 		},
// 		{
// 			setAsDefault:               true,
// 			runtimeName:                "NAME",
// 			expectedDefaultRuntimeName: "NAME",
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
// 			o := &container.Options{
// 				RuntimeName:  tc.runtimeName,
// 				RuntimeDir:   runtimeDir,
// 				SetAsDefault: tc.setAsDefault,
// 			}

// 			v2, err := containerd.New(
// 				containerd.WithLogger(logger),
// 				containerd.WithConfigSource(toml.Empty),
// 				containerd.WithRuntimeType(runtimeType),
// 				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
// 			)
// 			require.NoError(t, err)

// 			err = o.UpdateConfig(v2)
// 			require.NoError(t, err)

// 			cfg := v2.(*containerd.Config)

// 			defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
// 			require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName)
// 		})
// 	}
// }

// func TestUpdateV2Config(t *testing.T) {
// 	logger, _ := testlog.NewNullLogger()
// 	const runtimeDir = "/test/runtime/dir"

// 	testCases := []struct {
// 		runtimeName    string
// 		expectedConfig map[string]interface{}
// 	}{
// 		{
// 			runtimeName: "nvidia",
// 			expectedConfig: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"nvidia": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime",
// 									},
// 								},
// 								"nvidia-cdi": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.cdi",
// 									},
// 								},
// 								"nvidia-legacy": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.legacy",
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			runtimeName: "NAME",
// 			expectedConfig: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"NAME": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime",
// 									},
// 								},
// 								"nvidia-cdi": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.cdi",
// 									},
// 								},
// 								"nvidia-legacy": map[string]interface{}{
// 									"runtime_type":                    "runtime_type",
// 									"runtime_root":                    "",
// 									"runtime_engine":                  "",
// 									"privileged_without_host_devices": false,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"BinaryName": "/test/runtime/dir/nvidia-container-runtime.legacy",
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
// 			o := &container.Options{
// 				RuntimeName: tc.runtimeName,
// 				RuntimeDir:  runtimeDir,
// 			}

// 			v2, err := containerd.New(
// 				containerd.WithLogger(logger),
// 				containerd.WithConfigSource(toml.Empty),
// 				containerd.WithRuntimeType(runtimeType),
// 				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
// 			)
// 			require.NoError(t, err)

// 			err = o.UpdateConfig(v2)
// 			require.NoError(t, err)

// 			expected, err := toml.TreeFromMap(tc.expectedConfig)
// 			require.NoError(t, err)

// 			require.Equal(t, expected.String(), v2.String())
// 		})
// 	}

// }

// func TestUpdateV2ConfigWithRuncPresent(t *testing.T) {
// 	logger, _ := testlog.NewNullLogger()
// 	const runtimeDir = "/test/runtime/dir"

// 	testCases := []struct {
// 		runtimeName    string
// 		expectedConfig map[string]interface{}
// 	}{
// 		{
// 			runtimeName: "nvidia",
// 			expectedConfig: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"runc": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/runc-binary",
// 									},
// 								},
// 								"nvidia": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime",
// 									},
// 								},
// 								"nvidia-cdi": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.cdi",
// 									},
// 								},
// 								"nvidia-legacy": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.legacy",
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			runtimeName: "NAME",
// 			expectedConfig: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"runc": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/runc-binary",
// 									},
// 								},
// 								"NAME": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime",
// 									},
// 								},
// 								"nvidia-cdi": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.cdi",
// 									},
// 								},
// 								"nvidia-legacy": map[string]interface{}{
// 									"runtime_type":                    "runc_runtime_type",
// 									"runtime_root":                    "runc_runtime_root",
// 									"runtime_engine":                  "runc_runtime_engine",
// 									"privileged_without_host_devices": true,
// 									"container_annotations":           []string{"cdi.k8s.io/*"},
// 									"options": map[string]interface{}{
// 										"runc-option": "value",
// 										"BinaryName":  "/test/runtime/dir/nvidia-container-runtime.legacy",
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
// 			o := &container.Options{
// 				RuntimeName: tc.runtimeName,
// 				RuntimeDir:  runtimeDir,
// 			}

// 			v2, err := containerd.New(
// 				containerd.WithLogger(logger),
// 				containerd.WithConfigSource(toml.FromMap(runcConfigMapV2("/runc-binary"))),
// 				containerd.WithRuntimeType(runtimeType),
// 				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
// 			)
// 			require.NoError(t, err)

// 			err = o.UpdateConfig(v2)
// 			require.NoError(t, err)

// 			expected, err := toml.TreeFromMap(tc.expectedConfig)
// 			require.NoError(t, err)

// 			require.Equal(t, expected.String(), v2.String())
// 		})
// 	}
// }

// func TestUpdateV2ConfigEnableCDI(t *testing.T) {
// 	logger, _ := testlog.NewNullLogger()
// 	const runtimeDir = "/test/runtime/dir"

// 	testCases := []struct {
// 		enableCDI              bool
// 		expectedEnableCDIValue interface{}
// 	}{
// 		{},
// 		{
// 			enableCDI:              false,
// 			expectedEnableCDIValue: nil,
// 		},
// 		{
// 			enableCDI:              true,
// 			expectedEnableCDIValue: true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(fmt.Sprintf("%v", tc.enableCDI), func(t *testing.T) {
// 			o := &container.Options{
// 				EnableCDI:    tc.enableCDI,
// 				RuntimeName:  "nvidia",
// 				RuntimeDir:   runtimeDir,
// 				SetAsDefault: false,
// 			}

// 			cfg, err := toml.LoadMap(map[string]interface{}{})
// 			require.NoError(t, err)

// 			v2 := &containerd.Config{
// 				Logger:               logger,
// 				Tree:                 cfg,
// 				RuntimeType:          runtimeType,
// 				CRIRuntimePluginName: "io.containerd.grpc.v1.cri",
// 			}

// 			err = o.UpdateConfig(v2)
// 			require.NoError(t, err)

// 			enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
// 			require.EqualValues(t, tc.expectedEnableCDIValue, enableCDIValue)
// 		})
// 	}
// }

// func TestRevertV2Config(t *testing.T) {
// 	logger, _ := testlog.NewNullLogger()

// 	testCases := []struct {
// 		config map[string]interface {
// 		}
// 		expected map[string]interface{}
// 	}{
// 		{},
// 		{
// 			config: map[string]interface{}{
// 				"version": int64(2),
// 			},
// 		},
// 		{
// 			config: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"nvidia": runtimeMapV2("/test/runtime/dir/nvidia-container-runtime"),
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			config: map[string]interface{}{
// 				"version": int64(2),
// 				"plugins": map[string]interface{}{
// 					"io.containerd.grpc.v1.cri": map[string]interface{}{
// 						"containerd": map[string]interface{}{
// 							"runtimes": map[string]interface{}{
// 								"nvidia": runtimeMapV2("/test/runtime/dir/nvidia-container-runtime"),
// 							},
// 							"default_runtime_name": "nvidia",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
// 			o := &container.Options{
// 				RuntimeName: "nvidia",
// 			}

// 			expected, err := toml.TreeFromMap(tc.expected)
// 			require.NoError(t, err)

// 			v2, err := containerd.New(
// 				containerd.WithLogger(logger),
// 				containerd.WithConfigSource(toml.FromMap(tc.config)),
// 				containerd.WithRuntimeType(runtimeType),
// 				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
// 			)
// 			require.NoError(t, err)

// 			err = o.RevertConfig(v2)
// 			require.NoError(t, err)

// 			require.Equal(t, expected.String(), v2.String())
// 		})
// 	}
// }

// func runtimeMapV2(binary string) map[string]interface{} {
// 	return map[string]interface{}{
// 		"runtime_type":                    runtimeType,
// 		"runtime_root":                    "",
// 		"runtime_engine":                  "",
// 		"privileged_without_host_devices": false,
// 		"options": map[string]interface{}{
// 			"BinaryName": binary,
// 		},
// 	}
// }

// func runcConfigMapV2(binary string) map[string]interface{} {
// 	return map[string]interface{}{
// 		"version": 2,
// 		"plugins": map[string]interface{}{
// 			"io.containerd.grpc.v1.cri": map[string]interface{}{
// 				"containerd": map[string]interface{}{
// 					"runtimes": map[string]interface{}{
// 						"runc": map[string]interface{}{
// 							"runtime_type":                    "runc_runtime_type",
// 							"runtime_root":                    "runc_runtime_root",
// 							"runtime_engine":                  "runc_runtime_engine",
// 							"privileged_without_host_devices": true,
// 							"options": map[string]interface{}{
// 								"runc-option": "value",
// 								"BinaryName":  binary,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }
