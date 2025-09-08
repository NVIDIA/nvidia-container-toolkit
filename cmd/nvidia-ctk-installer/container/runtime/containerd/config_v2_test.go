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

const (
	runtimeType = "runtime_type"
	runtimeDir  = "/test/runtime/dir"
)

// TestUpdateV2Config_NoConfigFile tests the scenario when there is no
// containerd config file present
func TestUpdateV2Config_NoConfigFile(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		name         string
		runtimeName  string
		enableCDI    bool
		setAsDefault bool
		withRunc     bool
	}{
		{
			name:         "basic nvidia runtime",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "nvidia runtime with CDI enabled",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "nvidia runtime as default",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "nvidia runtime with CDI and as default",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "custom runtime name",
			runtimeName:  "CUSTOM",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "custom runtime with CDI and as default",
			runtimeName:  "CUSTOM",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "nvidia runtime with runc present",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     true,
		},
		{
			name:         "nvidia runtime with runc, CDI and as default",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := &container.Options{
				RuntimeName:  tc.runtimeName,
				RuntimeDir:   runtimeDir,
				EnableCDI:    tc.enableCDI,
				SetAsDefault: tc.setAsDefault,
			}

			var configSource toml.Loader
			if tc.withRunc {
				configSource = toml.FromMap(runcConfigMapV2("/runc-binary"))
			} else {
				configSource = toml.Empty
			}

			v2, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(configSource),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			cfg := v2.(*containerd.Config)

			// Verify the runtime was added
			addedRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", tc.runtimeName})
			require.NotNil(t, addedRuntime)

			// Verify nvidia-cdi and nvidia-legacy runtimes were added
			nvidiaLegacy := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia-legacy"})
			require.NotNil(t, nvidiaLegacy)
			nvidiaCDI := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia-cdi"})
			require.NotNil(t, nvidiaCDI)

			// Verify CDI enablement
			if tc.enableCDI {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.True(t, enableCDIValue.(bool))
			} else {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.Nil(t, enableCDIValue)
			}

			// Verify default runtime
			if tc.setAsDefault {
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Equal(t, tc.runtimeName, defaultRuntimeName)
			} else {
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Nil(t, defaultRuntimeName)
			}

			// Verify runc config inheritance
			if tc.withRunc {
				// Convert *toml.Tree to map for easier assertions
				runtimeTree, ok := addedRuntime.(*toml.Tree)
				if ok {
					runtimeMap := runtimeTree.ToMap()
					require.Equal(t, "runc_runtime_type", runtimeMap["runtime_type"])
					require.Equal(t, "runc_runtime_root", runtimeMap["runtime_root"])
					require.Equal(t, "runc_runtime_engine", runtimeMap["runtime_engine"])
					require.True(t, runtimeMap["privileged_without_host_devices"].(bool))
					options := runtimeMap["options"].(map[string]interface{})
					require.Equal(t, "value", options["runc-option"])
				} else {
					// If it's already a map
					runtimeMap := addedRuntime.(map[string]interface{})
					require.Equal(t, "runc_runtime_type", runtimeMap["runtime_type"])
					require.Equal(t, "runc_runtime_root", runtimeMap["runtime_root"])
					require.Equal(t, "runc_runtime_engine", runtimeMap["runtime_engine"])
					require.True(t, runtimeMap["privileged_without_host_devices"].(bool))
					options := runtimeMap["options"].(map[string]interface{})
					require.Equal(t, "value", options["runc-option"])
				}
			}
		})
	}
}

// TestUpdateV2Config_ExistingConfigWithoutNvidia tests the scenario when there
// is an existing config file without nvidia entries
func TestUpdateV2Config_ExistingConfigWithoutNvidia(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	existingConfigWithoutNvidia := map[string]interface{}{
		"version": int64(2),
		"plugins": map[string]interface{}{
			"io.containerd.grpc.v1.cri": map[string]interface{}{
				"containerd": map[string]interface{}{
					"default_runtime_name": "my-default",
					"runtimes": map[string]interface{}{
						"my-default": map[string]interface{}{
							"runtime_type": "io.containerd.runc.v2",
							"options": map[string]interface{}{
								"BinaryName": "/usr/bin/my-runtime",
							},
						},
					},
				},
				"registry": map[string]interface{}{
					"mirrors": map[string]interface{}{
						"docker.io": map[string]interface{}{
							"endpoint": []string{"https://registry-1.docker.io"},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name         string
		runtimeName  string
		enableCDI    bool
		setAsDefault bool
		withRunc     bool
	}{
		{
			name:         "add nvidia runtime to existing config",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "add nvidia runtime with CDI to existing config",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "add nvidia runtime as default to existing config",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "add custom runtime with all features to existing config",
			runtimeName:  "gpu-runtime",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "add nvidia runtime to existing config with runc",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     true,
		},
		{
			name:         "add nvidia runtime with all features and runc",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := &container.Options{
				RuntimeName:  tc.runtimeName,
				RuntimeDir:   runtimeDir,
				EnableCDI:    tc.enableCDI,
				SetAsDefault: tc.setAsDefault,
			}

			// Create a deep copy of the existing config
			configMap := deepCopyMap(existingConfigWithoutNvidia)
			if tc.withRunc {
				// Add runc runtime to the existing config
				runtimes := configMap["plugins"].(map[string]interface{})["io.containerd.grpc.v1.cri"].(map[string]interface{})["containerd"].(map[string]interface{})["runtimes"].(map[string]interface{})
				runtimes["runc"] = createRuncRuntimeConfig("/runc-binary")
			}

			v2, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.FromMap(configMap)),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			cfg := v2.(*containerd.Config)

			// Verify the existing config is preserved
			existingRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "my-default"})
			require.NotNil(t, existingRuntime)

			registry := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "registry"})
			require.NotNil(t, registry)

			// Verify the nvidia runtime was added
			addedRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", tc.runtimeName})
			require.NotNil(t, addedRuntime)

			// Verify nvidia-cdi and nvidia-legacy runtimes were added
			nvidiaLegacy := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia-legacy"})
			require.NotNil(t, nvidiaLegacy)
			nvidiaCDI := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia-cdi"})
			require.NotNil(t, nvidiaCDI)

			// Verify CDI enablement
			if tc.enableCDI {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.True(t, enableCDIValue.(bool))
			} else {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.Nil(t, enableCDIValue)
			}

			// Verify default runtime
			if tc.setAsDefault {
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Equal(t, tc.runtimeName, defaultRuntimeName)
			} else {
				// Should preserve the existing default
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Equal(t, "my-default", defaultRuntimeName)
			}

			// Verify runtime inheritance
			if tc.withRunc {
				// Convert *toml.Tree to map for easier assertions
				runtimeTree, ok := addedRuntime.(*toml.Tree)
				if ok {
					runtimeMap := runtimeTree.ToMap()
					require.NotNil(t, runtimeMap["runtime_type"])
					require.NotNil(t, runtimeMap["options"])
				} else {
					// If it's already a map
					runtimeMap := addedRuntime.(map[string]interface{})
					require.NotNil(t, runtimeMap["runtime_type"])
					require.NotNil(t, runtimeMap["options"])
				}
			}
		})
	}
}

// TestUpdateV2Config_ExistingConfigWithNvidia tests the scenario when there is
// an existing config file with nvidia entries
func TestUpdateV2Config_ExistingConfigWithNvidia(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	existingConfigWithNvidia := map[string]interface{}{
		"version": int64(2),
		"plugins": map[string]interface{}{
			"io.containerd.grpc.v1.cri": map[string]interface{}{
				"containerd": map[string]interface{}{
					"default_runtime_name": "my-default",
					"runtimes": map[string]interface{}{
						"my-default": map[string]interface{}{
							"runtime_type": "io.containerd.runc.v2",
							"options": map[string]interface{}{
								"BinaryName": "/usr/bin/my-runtime",
							},
						},
						"nvidia": map[string]interface{}{
							"runtime_type": "old-runtime-type",
							"runtime_root": "old-runtime-root",
							"options": map[string]interface{}{
								"BinaryName": "/old/nvidia/runtime",
							},
						},
						"nvidia-cdi": map[string]interface{}{
							"runtime_type": "old-cdi-type",
						},
						"nvidia-legacy": map[string]interface{}{
							"runtime_type": "old-legacy-type",
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name         string
		runtimeName  string
		enableCDI    bool
		setAsDefault bool
		withRunc     bool
	}{
		{
			name:         "update existing nvidia runtime",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "update nvidia runtime with CDI",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: false,
			withRunc:     false,
		},
		{
			name:         "update nvidia runtime as default",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "add custom runtime alongside existing nvidia",
			runtimeName:  "gpu-runtime",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     false,
		},
		{
			name:         "update with runc present",
			runtimeName:  "nvidia",
			enableCDI:    false,
			setAsDefault: false,
			withRunc:     true,
		},
		{
			name:         "update all features with runc",
			runtimeName:  "nvidia",
			enableCDI:    true,
			setAsDefault: true,
			withRunc:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := &container.Options{
				RuntimeName:  tc.runtimeName,
				RuntimeDir:   runtimeDir,
				EnableCDI:    tc.enableCDI,
				SetAsDefault: tc.setAsDefault,
			}

			// Create a deep copy of the existing config
			configMap := deepCopyMap(existingConfigWithNvidia)
			if tc.withRunc {
				// Add runc runtime to the existing config
				runtimes := configMap["plugins"].(map[string]interface{})["io.containerd.grpc.v1.cri"].(map[string]interface{})["containerd"].(map[string]interface{})["runtimes"].(map[string]interface{})
				runtimes["runc"] = createRuncRuntimeConfig("/runc-binary")
			}

			v2, err := containerd.New(
				containerd.WithLogger(logger),
				containerd.WithConfigSource(toml.FromMap(configMap)),
				containerd.WithRuntimeType(runtimeType),
				containerd.WithContainerAnnotations("cdi.k8s.io/*"),
			)
			require.NoError(t, err)

			err = o.UpdateConfig(v2)
			require.NoError(t, err)

			cfg := v2.(*containerd.Config)

			// Verify the existing config is preserved
			existingRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "my-default"})
			require.NotNil(t, existingRuntime)

			// Verify the runtime was added/updated
			addedRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", tc.runtimeName})
			require.NotNil(t, addedRuntime)

			// For the nvidia runtime, check that it was updated with proper options
			if tc.runtimeName == "nvidia" {
				runtimeTree, ok := addedRuntime.(*toml.Tree)
				if ok {
					runtimeMap := runtimeTree.ToMap()
					// Should have the configured runtime type (inherits from my-default runtime)
					require.Equal(t, "io.containerd.runc.v2", runtimeMap["runtime_type"])
					// Should have the BinaryName option set
					options := runtimeMap["options"].(map[string]interface{})
					require.Equal(t, fmt.Sprintf("%s/nvidia-container-runtime", runtimeDir), options["BinaryName"])
				} else {
					runtimeMap := addedRuntime.(map[string]interface{})
					require.Equal(t, "io.containerd.runc.v2", runtimeMap["runtime_type"])
					options := runtimeMap["options"].(map[string]interface{})
					require.Equal(t, fmt.Sprintf("%s/nvidia-container-runtime", runtimeDir), options["BinaryName"])
				}
			}

			// Verify CDI enablement
			if tc.enableCDI {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.True(t, enableCDIValue.(bool))
			} else {
				enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
				require.Nil(t, enableCDIValue)
			}

			// Verify default runtime
			if tc.setAsDefault {
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Equal(t, tc.runtimeName, defaultRuntimeName)
			} else {
				// Should preserve the existing default
				defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
				require.Equal(t, "my-default", defaultRuntimeName)
			}

			// Verify runc runtime if added
			if tc.withRunc {
				runcRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "runc"})
				require.NotNil(t, runcRuntime)
				// Convert *toml.Tree to map for easier assertions
				runtimeTree, ok := runcRuntime.(*toml.Tree)
				if ok {
					runtimeMap := runtimeTree.ToMap()
					require.Equal(t, "runc_runtime_type", runtimeMap["runtime_type"])
					require.Equal(t, "runc_runtime_root", runtimeMap["runtime_root"])
					require.Equal(t, "runc_runtime_engine", runtimeMap["runtime_engine"])
					require.True(t, runtimeMap["privileged_without_host_devices"].(bool))
					options := runtimeMap["options"].(map[string]interface{})
					require.Equal(t, "value", options["runc-option"])
				}
			}
		})
	}
}

// TestUpdateV2Config_InvalidDefaultRuntime tests the behavior when setting
// nvidia runtime as default alongside an existing runc configuration
func TestUpdateV2Config_InvalidDefaultRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	configWithRunc := map[string]interface{}{
		"version": int64(2),
		"plugins": map[string]interface{}{
			"io.containerd.grpc.v1.cri": map[string]interface{}{
				"containerd": map[string]interface{}{
					"default_runtime_name": "runc",
					"runtimes": map[string]interface{}{
						"runc": map[string]interface{}{
							"runtime_type": "io.containerd.runc.v2",
							"options": map[string]interface{}{
								"BinaryName": "/usr/bin/runc",
							},
						},
					},
				},
			},
		},
	}

	o := &container.Options{
		RuntimeName:  "nvidia",
		RuntimeDir:   runtimeDir,
		EnableCDI:    true,
		SetAsDefault: true,
	}

	v2, err := containerd.New(
		containerd.WithLogger(logger),
		containerd.WithConfigSource(toml.FromMap(configWithRunc)),
		containerd.WithRuntimeType(runtimeType),
		containerd.WithContainerAnnotations("cdi.k8s.io/*"),
	)
	require.NoError(t, err)

	err = o.UpdateConfig(v2)
	require.NoError(t, err)

	cfg := v2.(*containerd.Config)

	// Verify that nvidia runtime is now the default
	defaultRuntimeName := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
	require.Equal(t, "nvidia", defaultRuntimeName)

	// Verify that the runc runtime is still present
	runcRuntime := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "runc"})
	require.NotNil(t, runcRuntime)

	// Verify CDI is enabled
	enableCDIValue := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
	require.True(t, enableCDIValue.(bool))
}

// runcConfigMapV2 returns a map representing a containerd config with
// a runc runtime defined
func runcConfigMapV2(binary string) map[string]interface{} {
	return map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"io.containerd.grpc.v1.cri": map[string]interface{}{
				"containerd": map[string]interface{}{
					"runtimes": map[string]interface{}{
						"runc": createRuncRuntimeConfig(binary),
					},
				},
			},
		},
	}
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = deepCopyMap(val)
		case []interface{}:
			dst[k] = deepCopySlice(val)
		default:
			dst[k] = val
		}
	}
	return dst
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(src []interface{}) []interface{} {
	dst := make([]interface{}, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[i] = deepCopyMap(val)
		case []interface{}:
			dst[i] = deepCopySlice(val)
		default:
			dst[i] = val
		}
	}
	return dst
}

// createRuncRuntimeConfig creates a runc runtime configuration
// map with standard settings
func createRuncRuntimeConfig(binaryPath string) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type":                    "runc_runtime_type",
		"runtime_root":                    "runc_runtime_root",
		"runtime_engine":                  "runc_runtime_engine",
		"privileged_without_host_devices": true,
		"options": map[string]interface{}{
			"runc-option": "value",
			"BinaryName":  binaryPath,
		},
	}
}
