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
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestUpdateV1ConfigDefaultRuntime(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"

	testCases := []struct {
		legacyConfig                 bool
		setAsDefault                 bool
		runtimeClass                 string
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
			runtimeClass:                 "NAME",
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: "/test/runtime/dir/nvidia-container-runtime",
		},
		{
			legacyConfig:                 true,
			setAsDefault:                 true,
			runtimeClass:                 "nvidia-experimental",
			expectedDefaultRuntimeName:   nil,
			expectedDefaultRuntimeBinary: "/test/runtime/dir/nvidia-container-runtime-experimental",
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
			runtimeClass:                 "NAME",
			expectedDefaultRuntimeName:   "NAME",
			expectedDefaultRuntimeBinary: nil,
		},
		{
			legacyConfig:                 false,
			setAsDefault:                 true,
			runtimeClass:                 "nvidia-experimental",
			expectedDefaultRuntimeName:   "nvidia-experimental",
			expectedDefaultRuntimeBinary: nil,
		},
	}

	for i, tc := range testCases {
		o := &options{
			useLegacyConfig: tc.legacyConfig,
			setAsDefault:    tc.setAsDefault,
			runtimeClass:    tc.runtimeClass,
			runtimeType:     runtimeType,
			runtimeDir:      runtimeDir,
		}

		config, err := toml.TreeFromMap(map[string]interface{}{})
		require.NoError(t, err, "%d: %v", i, tc)

		err = UpdateV1Config(config, o)
		require.NoError(t, err, "%d: %v", i, tc)

		defaultRuntimeName := config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime_name"})
		require.EqualValues(t, tc.expectedDefaultRuntimeName, defaultRuntimeName, "%d: %v", i, tc)

		defaultRuntime := config.GetPath([]string{"plugins", "cri", "containerd", "default_runtime"})
		if tc.expectedDefaultRuntimeBinary == nil {
			require.Nil(t, defaultRuntime, "%d: %v", i, tc)
		} else {
			expected, err := defaultRuntimeTomlConfigV1(tc.expectedDefaultRuntimeBinary.(string))
			require.NoError(t, err, "%d: %v", i, tc)

			configContents, _ := toml.Marshal(defaultRuntime.(*toml.Tree))
			expectedContents, _ := toml.Marshal(expected)

			require.Equal(t, string(expectedContents), string(configContents), "%d: %v: %v", i, tc)
		}

	}
}

func TestUpdateV1Config(t *testing.T) {
	const runtimeDir = "/test/runtime/dir"
	const expectedVersion = int64(1)

	expectedBinaries := []string{
		"/test/runtime/dir/nvidia-container-runtime",
		"/test/runtime/dir/nvidia-container-runtime-experimental",
	}

	testCases := []struct {
		runtimeClass     string
		expectedRuntimes []string
	}{
		{
			runtimeClass:     "nvidia",
			expectedRuntimes: []string{"nvidia", "nvidia-experimental"},
		},
		{
			runtimeClass:     "NAME",
			expectedRuntimes: []string{"NAME", "nvidia-experimental"},
		},
		{
			runtimeClass:     "nvidia-experimental",
			expectedRuntimes: []string{"nvidia", "nvidia-experimental"},
		},
	}

	for i, tc := range testCases {
		o := &options{
			runtimeClass: tc.runtimeClass,
			runtimeType:  runtimeType,
			runtimeDir:   runtimeDir,
		}

		config, err := toml.TreeFromMap(map[string]interface{}{})
		require.NoError(t, err, "%d: %v", i, tc)

		err = UpdateV1Config(config, o)
		require.NoError(t, err, "%d: %v", i, tc)

		version, ok := config.Get("version").(int64)
		require.True(t, ok)
		require.EqualValues(t, expectedVersion, version)

		runtimes, ok := config.GetPath([]string{"plugins", "cri", "containerd", "runtimes"}).(*toml.Tree)
		require.True(t, ok)

		runtimeClasses := runtimes.Keys()
		require.ElementsMatch(t, tc.expectedRuntimes, runtimeClasses, "%d: %v", i, tc)

		for i, r := range tc.expectedRuntimes {
			runtimeConfig := runtimes.Get(r)

			expected, err := runtimeTomlConfigV1(expectedBinaries[i])
			require.NoError(t, err, "%d: %v", i, tc)

			configContents, _ := toml.Marshal(runtimeConfig)
			expectedContents, _ := toml.Marshal(expected)

			require.Equal(t, string(expectedContents), string(configContents), "%d: %v: %v", i, r, tc)

		}
	}
}

func TestUpdateV1ConfigWithRuncPresent(t *testing.T) {
	const runcBinary = "/runc-binary"
	const runtimeDir = "/test/runtime/dir"
	const expectedVersion = int64(1)

	expectedBinaries := []string{
		runcBinary,
		"/test/runtime/dir/nvidia-container-runtime",
		"/test/runtime/dir/nvidia-container-runtime-experimental",
	}

	testCases := []struct {
		runtimeClass     string
		expectedRuntimes []string
	}{
		{
			runtimeClass:     "nvidia",
			expectedRuntimes: []string{"runc", "nvidia", "nvidia-experimental"},
		},
		{
			runtimeClass:     "NAME",
			expectedRuntimes: []string{"runc", "NAME", "nvidia-experimental"},
		},
		{
			runtimeClass:     "nvidia-experimental",
			expectedRuntimes: []string{"runc", "nvidia", "nvidia-experimental"},
		},
	}

	for i, tc := range testCases {
		o := &options{
			runtimeClass: tc.runtimeClass,
			runtimeType:  runtimeType,
			runtimeDir:   runtimeDir,
		}

		config, err := toml.TreeFromMap(runcConfigMapV1("/runc-binary"))
		require.NoError(t, err, "%d: %v", i, tc)

		err = UpdateV1Config(config, o)
		require.NoError(t, err, "%d: %v", i, tc)

		version, ok := config.Get("version").(int64)
		require.True(t, ok)
		require.EqualValues(t, expectedVersion, version)

		runtimes, ok := config.GetPath([]string{"plugins", "cri", "containerd", "runtimes"}).(*toml.Tree)
		require.True(t, ok)

		runtimeClasses := runtimes.Keys()
		require.ElementsMatch(t, tc.expectedRuntimes, runtimeClasses, "%d: %v", i, tc)

		for i, r := range tc.expectedRuntimes {
			runtimeConfig := runtimes.Get(r)

			expected, err := toml.TreeFromMap(runcRuntimeConfigMapV1(expectedBinaries[i]))
			require.NoError(t, err, "%d: %v", i, tc)

			configContents, _ := toml.Marshal(runtimeConfig)
			expectedContents, _ := toml.Marshal(expected)

			require.Equal(t, string(expectedContents), string(configContents), "%d: %v: %v", i, r, tc)

		}
	}
}

func TestRevertV1Config(t *testing.T) {
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
								"nvidia":              runtimeMapV1("/test/runtime/dir/nvidia-container-runtime"),
								"nvidia-experimental": runtimeMapV1("/test/runtime/dir/nvidia-container-runtime-experimental"),
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
								"nvidia":              runtimeMapV1("/test/runtime/dir/nvidia-container-runtime"),
								"nvidia-experimental": runtimeMapV1("/test/runtime/dir/nvidia-container-runtime-experimental"),
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
		o := &options{
			runtimeClass: "nvidia",
		}

		config, err := toml.TreeFromMap(tc.config)
		require.NoError(t, err, "%d: %v", i, tc)

		expected, err := toml.TreeFromMap(tc.expected)
		require.NoError(t, err, "%d: %v", i, tc)

		err = RevertV1Config(config, o)
		require.NoError(t, err, "%d: %v", i, tc)

		configContents, _ := toml.Marshal(config)
		expectedContents, _ := toml.Marshal(expected)

		require.Equal(t, string(expectedContents), string(configContents), "%d: %v", i, tc)
	}
}

func runtimeTomlConfigV1(binary string) (*toml.Tree, error) {
	return toml.TreeFromMap(runtimeMapV1(binary))
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
						"runc": runcRuntimeConfigMapV1(binary),
					},
				},
			},
		},
	}
}

func runcRuntimeConfigMapV1(binary string) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type":                    "runc_runtime_type",
		"runtime_root":                    "runc_runtime_root",
		"runtime_engine":                  "runc_runtime_engine",
		"privileged_without_host_devices": true,
		"options": map[string]interface{}{
			"runc-option": "value",
			"BinaryName":  binary,
			"Runtime":     binary,
		},
	}
}
