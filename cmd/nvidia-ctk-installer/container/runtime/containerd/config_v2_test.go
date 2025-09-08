/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/containerd"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// TestSetupCleanup tests the complete Setup->Cleanup lifecycle following Evan Lezar's pattern
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
				Config:       "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				runtimeType: defaultRuntimeType,
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				require.FileExists(t, filepath.Join(testRoot, "etc/containerd/config.toml"))
				verifyRuntimesPresent(t, filepath.Join(testRoot, "etc/containerd/config.toml"),
					"nvidia", "nvidia-cdi", "nvidia-legacy")
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				// Verify that nvidia runtimes were removed
				if _, err := os.Stat(filepath.Join(testRoot, "etc/containerd/config.toml")); err == nil {
					verifyRuntimesAbsent(t, filepath.Join(testRoot, "etc/containerd/config.toml"),
						"nvidia", "nvidia-cdi", "nvidia-legacy")
				}
				return nil
			},
		},
		{
			description: "existing config without nvidia runtime",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				EnableCDI:    true,
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				runtimeType: defaultRuntimeType,
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := containerd.New(
					containerd.WithConfigSource(toml.FromMap(map[string]interface{}{
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
					})),
				)
				require.NoError(t, err)
				_, err = cfg.Save(configPath)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)
				verifyRuntimesPresent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				verifyCDIEnabled(t, configPath, true)
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				// Note: CDI state is not reverted in current implementation
				return nil
			},
		},
		{
			description: "existing config with nvidia runtime already present",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: true,
				RestartMode:  "none",
			},
			options: Options{
				runtimeType: defaultRuntimeType,
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := containerd.New(
					containerd.WithConfigSource(toml.FromMap(map[string]interface{}{
						"version": int64(2),
						"plugins": map[string]interface{}{
							"io.containerd.grpc.v1.cri": map[string]interface{}{
								"containerd": map[string]interface{}{
									"default_runtime_name": "nvidia",
									"runtimes": map[string]interface{}{
										"runc": map[string]interface{}{
											"runtime_type": "io.containerd.runc.v2",
											"options": map[string]interface{}{
												"BinaryName": "/usr/bin/runc",
											},
										},
										"nvidia": map[string]interface{}{
											"runtime_type": "io.containerd.runc.v2",
											"options": map[string]interface{}{
												"BinaryName": "/old/path/nvidia-container-runtime",
											},
										},
									},
								},
							},
						},
					})),
				)
				require.NoError(t, err)
				_, err = cfg.Save(configPath)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)

				cfg, err := containerd.New(
					containerd.WithPath(configPath),
					containerd.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*containerd.Config)

				// Verify runtime path was updated
				nvidiaPath := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd",
					"runtimes", "nvidia", "options", "BinaryName"})
				require.Equal(t, "/usr/bin/nvidia-container-runtime", nvidiaPath)

				// Verify default runtime
				defaultRuntime := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri",
					"containerd", "default_runtime_name"})
				require.Equal(t, "nvidia", defaultRuntime)

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")

				// Default runtime should be cleared
				cfg, err := containerd.New(
					containerd.WithPath(configPath),
					containerd.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*containerd.Config)
				defaultRuntime := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri",
					"containerd", "default_runtime_name"})
				require.Nil(t, defaultRuntime)

				return nil
			},
		},
		{
			description: "complex config with multiple plugins and settings",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/containerd/config.toml",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				EnableCDI:    true,
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				runtimeType: defaultRuntimeType,
				ContainerRuntimeModesCDIAnnotationPrefixes: []string{"cdi.k8s.io"},
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := containerd.New(
					containerd.WithConfigSource(toml.FromMap(map[string]interface{}{
						"version": int64(2),
						"root":    "/var/lib/containerd",
						"state":   "/run/containerd",
						"plugins": map[string]interface{}{
							"io.containerd.grpc.v1.cri": map[string]interface{}{
								"containerd": map[string]interface{}{
									"snapshotter":          "overlayfs",
									"default_runtime_name": "runc",
									"runtimes": map[string]interface{}{
										"runc": map[string]interface{}{
											"runtime_type": "io.containerd.runc.v2",
											"options": map[string]interface{}{
												"BinaryName":    "/usr/bin/runc",
												"SystemdCgroup": true,
											},
										},
										"custom": map[string]interface{}{
											"runtime_type": "io.containerd.custom.v1",
											"options": map[string]interface{}{
												"TypeUrl": "custom.runtime/options",
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
							"io.containerd.internal.v1.opt": map[string]interface{}{
								"path": "/opt/containerd",
							},
						},
					})),
				)
				require.NoError(t, err)
				_, err = cfg.Save(configPath)
				require.NoError(t, err)
				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)

				// Verify nvidia runtimes added
				verifyRuntimesPresent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				// Verify CDI enabled
				verifyCDIEnabled(t, configPath, true)

				// Verify other config preserved
				cfg, err := containerd.New(
					containerd.WithPath(configPath),
					containerd.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*containerd.Config)

				// Check non-runtime settings preserved
				require.Equal(t, "/var/lib/containerd", c.GetPath([]string{"root"}))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/containerd/config.toml")
				require.FileExists(t, configPath)

				// Verify nvidia runtimes removed
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")

				// Note: CDI state and other settings should be preserved but CDI is not currently reverted
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Update any paths as required
			testRoot := t.TempDir()
			tc.containerOptions.Config = strings.ReplaceAll(tc.containerOptions.Config, "{{ .testRoot }}", testRoot)

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

// verifyRuntimesPresent checks that expected runtimes exist in config
func verifyRuntimesPresent(t *testing.T, configPath string, runtimes ...string) {
	cfg, err := containerd.New(
		containerd.WithPath(configPath),
		containerd.WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	c := cfg.(*containerd.Config)
	for _, runtime := range runtimes {
		runtimeConfig := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", runtime})
		require.NotNil(t, runtimeConfig, "Runtime %s should be present", runtime)
	}
}

// verifyRuntimesAbsent checks that runtimes don't exist in config
func verifyRuntimesAbsent(t *testing.T, configPath string, runtimes ...string) {
	// If config file doesn't exist, all runtimes are absent
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	cfg, err := containerd.New(
		containerd.WithPath(configPath),
		containerd.WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	c := cfg.(*containerd.Config)
	for _, runtime := range runtimes {
		runtimeConfig := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", runtime})
		require.Nil(t, runtimeConfig, "Runtime %s should not be present", runtime)
	}
}

// verifyCDIEnabled checks if CDI is enabled in the config
func verifyCDIEnabled(t *testing.T, configPath string, expectedEnabled bool) {
	cfg, err := containerd.New(
		containerd.WithPath(configPath),
		containerd.WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	c := cfg.(*containerd.Config)
	enableCDI := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})

	if expectedEnabled {
		require.NotNil(t, enableCDI, "CDI should be enabled")
		require.True(t, enableCDI.(bool), "CDI should be set to true")
	} else if enableCDI != nil {
		// CDI can be either absent or false
		require.False(t, enableCDI.(bool), "CDI should be set to false")
	}
}
