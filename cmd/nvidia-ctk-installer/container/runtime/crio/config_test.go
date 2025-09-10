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

package crio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine/crio"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// TestSetupCleanupConfigMode tests the Setup->Cleanup lifecycle for config mode
func TestSetupCleanupConfigMode(t *testing.T) {
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
			description: "config mode: top-level config does not exist",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)
				verifyRuntimesPresent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoFileExists(t, configPath)
				return nil
			},
		},
		{
			description: "config mode: existing config without nvidia runtime",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := crio.New(
					crio.WithConfigSource(toml.FromMap(map[string]interface{}{
						"crio": map[string]interface{}{
							"runtime": map[string]interface{}{
								"default_runtime": "crun",
								"runtimes": map[string]interface{}{
									"crun": map[string]interface{}{
										"runtime_path": "/usr/bin/crun",
										"runtime_type": "oci",
										"runtime_root": "/run/crun",
										"monitor_path": "/usr/libexec/crio/conmon",
									},
								},
							},
							"image": map[string]interface{}{
								"signature_policy": "/etc/crio/policy.json",
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
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)
				verifyRuntimesPresent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy", "crun")

				// Verify image config preserved
				cfg, err := crio.New(
					crio.WithPath(configPath),
					crio.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*crio.Config)
				signaturePolicy := c.GetPath([]string{"crio", "image", "signature_policy"})
				require.Equal(t, "/etc/crio/policy.json", signaturePolicy)

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				verifyRuntimesPresent(t, configPath, "crun")
				return nil
			},
		},
		{
			description: "config mode: existing config with nvidia runtime already present",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: true,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := crio.New(
					crio.WithConfigSource(toml.FromMap(map[string]interface{}{
						"crio": map[string]interface{}{
							"runtime": map[string]interface{}{
								"default_runtime": "nvidia",
								"runtimes": map[string]interface{}{
									"crun": map[string]interface{}{
										"runtime_path": "/usr/bin/crun",
										"runtime_type": "oci",
									},
									"nvidia": map[string]interface{}{
										"runtime_path": "/old/path/nvidia-container-runtime",
										"runtime_type": "oci",
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
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				cfg, err := crio.New(
					crio.WithPath(configPath),
					crio.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*crio.Config)

				// Verify runtime path was updated
				nvidiaPath := c.GetPath([]string{"crio", "runtime", "runtimes", "nvidia", "runtime_path"})
				require.Equal(t, "/usr/bin/nvidia-container-runtime", nvidiaPath)

				// Verify default runtime
				defaultRuntime := c.GetPath([]string{"crio", "runtime", "default_runtime"})
				require.Equal(t, "nvidia", defaultRuntime)

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				verifyRuntimesPresent(t, configPath, "crun")

				// Default runtime should be cleared
				cfg, err := crio.New(
					crio.WithPath(configPath),
					crio.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*crio.Config)
				defaultRuntime := c.GetPath([]string{"crio", "runtime", "default_runtime"})
				require.Nil(t, defaultRuntime)

				return nil
			},
		},
		{
			description: "config mode: complex config with multiple settings",
			containerOptions: container.Options{
				Config:       "{{ .testRoot }}/etc/crio/crio.conf",
				RuntimeName:  "nvidia",
				RuntimeDir:   "/usr/bin",
				SetAsDefault: false,
				RestartMode:  "none",
			},
			options: Options{
				configMode: "config",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

				cfg, err := crio.New(
					crio.WithConfigSource(toml.FromMap(map[string]interface{}{
						"crio": map[string]interface{}{
							"runtime": map[string]interface{}{
								"default_runtime": "crun",
								"conmon":          "/usr/libexec/crio/conmon",
								"conmon_cgroup":   "pod",
								"selinux":         true,
								"runtimes": map[string]interface{}{
									"crun": map[string]interface{}{
										"runtime_path": "/usr/bin/crun",
										"runtime_type": "oci",
										"runtime_root": "/run/crun",
										"monitor_path": "/usr/libexec/crio/conmon",
									},
									"runc": map[string]interface{}{
										"runtime_path": "/usr/bin/runc",
										"runtime_type": "oci",
										"runtime_root": "/run/runc",
									},
								},
							},
							"image": map[string]interface{}{
								"signature_policy": "/etc/crio/policy.json",
								"insecure_registries": []string{
									"localhost:5000",
								},
							},
							"network": map[string]interface{}{
								"network_dir": "/etc/cni/net.d/",
								"plugin_dirs": []string{
									"/opt/cni/bin",
									"/usr/libexec/cni",
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
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				// Verify nvidia runtimes added
				verifyRuntimesPresent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				// Verify existing runtimes preserved
				verifyRuntimesPresent(t, configPath, "crun", "runc")

				// Verify other settings preserved
				cfg, err := crio.New(
					crio.WithPath(configPath),
					crio.WithConfigSource(toml.FromFile(configPath)),
				)
				require.NoError(t, err)
				c := cfg.(*crio.Config)

				// Check non-runtime settings
				require.Equal(t, "/usr/libexec/crio/conmon", c.GetPath([]string{"crio", "runtime", "conmon"}))
				require.Equal(t, true, c.GetPath([]string{"crio", "runtime", "selinux"}))
				require.Equal(t, "/etc/cni/net.d/", c.GetPath([]string{"crio", "network", "network_dir"}))

				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				configPath := filepath.Join(testRoot, "etc/crio/crio.conf")
				require.FileExists(t, configPath)

				// Verify nvidia runtimes removed
				verifyRuntimesAbsent(t, configPath, "nvidia", "nvidia-cdi", "nvidia-legacy")
				// Verify other runtimes preserved
				verifyRuntimesPresent(t, configPath, "crun", "runc")

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

// TestSetupCleanupHookMode tests the Setup->Cleanup lifecycle for hook mode
func TestSetupCleanupHookMode(t *testing.T) {
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
			description: "hook mode: no existing hook",
			containerOptions: container.Options{
				RuntimeName: "nvidia",
				RuntimeDir:  "/usr/bin",
				RestartMode: "none",
			},
			options: Options{
				configMode:   "hook",
				hooksDir:     "{{ .testRoot }}/etc/crio/hooks.d",
				hookFilename: "99-nvidia.json",
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				expectedBinary := filepath.Join("/usr/bin", config.NVIDIAContainerRuntimeHookExecutable)
				verifyHookExists(t, hookPath, expectedBinary)
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.NoFileExists(t, hookPath)
				return nil
			},
		},
		{
			description: "hook mode: existing hook file",
			containerOptions: container.Options{
				RuntimeName: "nvidia",
				RuntimeDir:  "/usr/bin",
				RestartMode: "none",
			},
			options: Options{
				configMode:   "hook",
				hooksDir:     "{{ .testRoot }}/etc/crio/hooks.d",
				hookFilename: "99-nvidia.json",
			},
			prepareEnvironment: func(t *testing.T, testRoot string) error {
				hooksDir := filepath.Join(testRoot, "etc/crio/hooks.d")
				require.NoError(t, os.MkdirAll(hooksDir, 0755))

				// Create existing hook with old path
				existingHook := map[string]interface{}{
					"version": "1.0.0",
					"hook": map[string]interface{}{
						"path": "/old/path/nvidia-container-runtime-hook",
						"args": []string{"nvidia-container-runtime-hook", "prestart"},
					},
					"when": map[string]interface{}{
						"always": true,
					},
					"stages": []string{"prestart"},
				}

				existingData, err := json.MarshalIndent(existingHook, "", "  ")
				require.NoError(t, err)

				hookPath := filepath.Join(hooksDir, "99-nvidia.json")
				err = os.WriteFile(hookPath, existingData, 0600)
				require.NoError(t, err)

				return nil
			},
			assertSetupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				expectedBinary := filepath.Join("/usr/bin", config.NVIDIAContainerRuntimeHookExecutable)
				verifyHookExists(t, hookPath, expectedBinary)
				return nil
			},
			assertCleanupPostConditions: func(t *testing.T, testRoot string) error {
				hookPath := filepath.Join(testRoot, "etc/crio/hooks.d/99-nvidia.json")
				require.NoFileExists(t, hookPath)
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Update any paths as required
			testRoot := t.TempDir()
			tc.options.hooksDir = strings.ReplaceAll(tc.options.hooksDir, "{{ .testRoot }}", testRoot)

			// Prepare the test environment
			if tc.prepareEnvironment != nil {
				require.NoError(t, tc.prepareEnvironment(t, testRoot))
			} else {
				// Ensure hooks directory exists
				hooksDir := strings.ReplaceAll(tc.options.hooksDir, "{{ .testRoot }}", testRoot)
				require.NoError(t, os.MkdirAll(hooksDir, 0755))
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
	cfg, err := crio.New(
		crio.WithPath(configPath),
		crio.WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	c := cfg.(*crio.Config)
	for _, runtime := range runtimes {
		runtimeConfig := c.GetPath([]string{"crio", "runtime", "runtimes", runtime})
		require.NotNil(t, runtimeConfig, "Runtime %s should be present", runtime)
	}
}

// verifyRuntimesAbsent checks that runtimes don't exist in config
func verifyRuntimesAbsent(t *testing.T, configPath string, runtimes ...string) {
	// If config file doesn't exist, all runtimes are absent
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	cfg, err := crio.New(
		crio.WithPath(configPath),
		crio.WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	c := cfg.(*crio.Config)
	for _, runtime := range runtimes {
		runtimeConfig := c.GetPath([]string{"crio", "runtime", "runtimes", runtime})
		require.Nil(t, runtimeConfig, "Runtime %s should not be present", runtime)
	}
}

// verifyHookExists checks if a hook file exists with expected content
func verifyHookExists(t *testing.T, hookPath string, expectedBinary string) {
	require.FileExists(t, hookPath)

	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)

	var hook map[string]interface{}
	err = json.Unmarshal(data, &hook)
	require.NoError(t, err)

	// Verify hook structure
	require.Equal(t, "1.0.0", hook["version"])
	hookObj := hook["hook"].(map[string]interface{})
	require.Equal(t, expectedBinary, hookObj["path"])
}
