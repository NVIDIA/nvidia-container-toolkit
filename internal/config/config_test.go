/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetConfigWithCustomConfig(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// By default debug is disabled
	contents := []byte("[nvidia-container-runtime]\ndebug = \"/nvidia-container-toolkit.log\"")
	testDir := filepath.Join(wd, "test")
	filename := filepath.Join(testDir, configFilePath)

	os.Setenv(configOverride, testDir)

	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0766))
	require.NoError(t, os.WriteFile(filename, contents, 0766))

	defer func() { require.NoError(t, os.RemoveAll(testDir)) }()

	cfg, err := GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg.NVIDIAContainerRuntimeConfig.DebugFilePath, "/nvidia-container-toolkit.log")
}

func TestGetConfig(t *testing.T) {
	testCases := []struct {
		description     string
		contents        []string
		expectedError   error
		inspectLdconfig bool
		expectedConfig  *Config
	}{
		{
			description:     "empty config is default",
			inspectLdconfig: true,
			expectedConfig: &Config{
				AcceptEnvvarUnprivileged: true,
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root:      "",
					LoadKmods: true,
					Ldconfig:  "WAS_CHECKED",
				},
				NVIDIAContainerRuntimeConfig: RuntimeConfig{
					DebugFilePath: "/dev/null",
					LogLevel:      "info",
					Runtimes:      []string{"docker-runc", "runc"},
					Mode:          "auto",
					Modes: modesConfig{
						CSV: csvModeConfig{
							MountSpecPath: "/etc/nvidia-container-runtime/host-files-for-container.d",
						},
						CDI: cdiModeConfig{
							DefaultKind:        "nvidia.com/gpu",
							AnnotationPrefixes: []string{"cdi.k8s.io/"},
							SpecDirs:           []string{"/etc/cdi", "/var/run/cdi"},
						},
					},
				},
				NVIDIAContainerRuntimeHookConfig: RuntimeHookConfig{
					Path: "nvidia-container-runtime-hook",
				},
				NVIDIACTKConfig: CTKConfig{
					Path: "nvidia-ctk",
				},
			},
		},
		{
			description: "config options set inline",
			contents: []string{
				"accept-nvidia-visible-devices-envvar-when-unprivileged = false",
				"nvidia-container-cli.root = \"/bar/baz\"",
				"nvidia-container-cli.load-kmods = false",
				"nvidia-container-cli.ldconfig = \"/foo/bar/ldconfig\"",
				"nvidia-container-runtime.debug = \"/foo/bar\"",
				"nvidia-container-runtime.discover-mode = \"not-legacy\"",
				"nvidia-container-runtime.log-level = \"debug\"",
				"nvidia-container-runtime.runtimes = [\"/some/runtime\",]",
				"nvidia-container-runtime.mode = \"not-auto\"",
				"nvidia-container-runtime.modes.cdi.default-kind = \"example.vendor.com/device\"",
				"nvidia-container-runtime.modes.cdi.annotation-prefixes = [\"cdi.k8s.io/\", \"example.vendor.com/\",]",
				"nvidia-container-runtime.modes.cdi.spec-dirs = [\"/except/etc/cdi\", \"/not/var/run/cdi\",]",
				"nvidia-container-runtime.modes.csv.mount-spec-path = \"/not/etc/nvidia-container-runtime/host-files-for-container.d\"",
				"nvidia-container-runtime-hook.path = \"/foo/bar/nvidia-container-runtime-hook\"",
				"nvidia-ctk.path = \"/foo/bar/nvidia-ctk\"",
			},
			expectedConfig: &Config{
				AcceptEnvvarUnprivileged: false,
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root:      "/bar/baz",
					LoadKmods: false,
					Ldconfig:  "/foo/bar/ldconfig",
				},
				NVIDIAContainerRuntimeConfig: RuntimeConfig{
					DebugFilePath: "/foo/bar",
					LogLevel:      "debug",
					Runtimes:      []string{"/some/runtime"},
					Mode:          "not-auto",
					Modes: modesConfig{
						CSV: csvModeConfig{
							MountSpecPath: "/not/etc/nvidia-container-runtime/host-files-for-container.d",
						},
						CDI: cdiModeConfig{
							DefaultKind: "example.vendor.com/device",
							AnnotationPrefixes: []string{
								"cdi.k8s.io/",
								"example.vendor.com/",
							},
							SpecDirs: []string{
								"/except/etc/cdi",
								"/not/var/run/cdi",
							},
						},
					},
				},
				NVIDIAContainerRuntimeHookConfig: RuntimeHookConfig{
					Path: "/foo/bar/nvidia-container-runtime-hook",
				},
				NVIDIACTKConfig: CTKConfig{
					Path: "/foo/bar/nvidia-ctk",
				},
			},
		},
		{
			description: "config options set in section",
			contents: []string{
				"accept-nvidia-visible-devices-envvar-when-unprivileged = false",
				"[nvidia-container-cli]",
				"root = \"/bar/baz\"",
				"load-kmods = false",
				"ldconfig = \"/foo/bar/ldconfig\"",
				"[nvidia-container-runtime]",
				"debug = \"/foo/bar\"",
				"discover-mode = \"not-legacy\"",
				"log-level = \"debug\"",
				"runtimes = [\"/some/runtime\",]",
				"mode = \"not-auto\"",
				"[nvidia-container-runtime.modes.cdi]",
				"default-kind = \"example.vendor.com/device\"",
				"annotation-prefixes = [\"cdi.k8s.io/\", \"example.vendor.com/\",]",
				"spec-dirs = [\"/except/etc/cdi\", \"/not/var/run/cdi\",]",
				"[nvidia-container-runtime.modes.csv]",
				"mount-spec-path = \"/not/etc/nvidia-container-runtime/host-files-for-container.d\"",
				"[nvidia-container-runtime-hook]",
				"path = \"/foo/bar/nvidia-container-runtime-hook\"",
				"[nvidia-ctk]",
				"path = \"/foo/bar/nvidia-ctk\"",
			},
			expectedConfig: &Config{
				AcceptEnvvarUnprivileged: false,
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root:      "/bar/baz",
					LoadKmods: false,
					Ldconfig:  "/foo/bar/ldconfig",
				},
				NVIDIAContainerRuntimeConfig: RuntimeConfig{
					DebugFilePath: "/foo/bar",
					LogLevel:      "debug",
					Runtimes:      []string{"/some/runtime"},
					Mode:          "not-auto",
					Modes: modesConfig{
						CSV: csvModeConfig{
							MountSpecPath: "/not/etc/nvidia-container-runtime/host-files-for-container.d",
						},
						CDI: cdiModeConfig{
							DefaultKind: "example.vendor.com/device",
							AnnotationPrefixes: []string{
								"cdi.k8s.io/",
								"example.vendor.com/",
							},
							SpecDirs: []string{
								"/except/etc/cdi",
								"/not/var/run/cdi",
							},
						},
					},
				},
				NVIDIAContainerRuntimeHookConfig: RuntimeHookConfig{
					Path: "/foo/bar/nvidia-container-runtime-hook",
				},
				NVIDIACTKConfig: CTKConfig{
					Path: "/foo/bar/nvidia-ctk",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			reader := strings.NewReader(strings.Join(tc.contents, "\n"))

			cfg, err := LoadFrom(reader)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// We first handle the ldconfig path since this is currently system-dependent.
			if tc.inspectLdconfig {
				ldconfig := cfg.NVIDIAContainerCLIConfig.Ldconfig
				require.True(t, strings.HasPrefix(ldconfig, "@/sbin/ldconfig"))
				remaining := strings.TrimPrefix(ldconfig, "@/sbin/ldconfig")
				require.True(t, remaining == ".real" || remaining == "")

				cfg.NVIDIAContainerCLIConfig.Ldconfig = "WAS_CHECKED"
			}

			require.EqualValues(t, tc.expectedConfig, cfg)
		})
	}
}

func TestConfigDefault(t *testing.T) {
	config, err := getDefault()
	require.NoError(t, err)

	buffer := new(bytes.Buffer)
	_, err = config.Save(buffer)
	require.NoError(t, err)

	var lines []string
	for _, l := range strings.Split(buffer.String(), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "# ") {
			l = "#" + strings.TrimPrefix(l, "# ")
		}
		lines = append(lines, l)
	}

	// We take the lines from the config that was included in previous packages.
	expectedLines := []string{
		"disable-require = false",
		"#swarm-resource = \"DOCKER_RESOURCE_GPU\"",
		"#accept-nvidia-visible-devices-envvar-when-unprivileged = true",
		"#accept-nvidia-visible-devices-as-volume-mounts = false",

		"#root = \"/run/nvidia/driver\"",
		"#path = \"/usr/bin/nvidia-container-cli\"",
		"environment = []",
		"#debug = \"/var/log/nvidia-container-toolkit.log\"",
		"#ldcache = \"/etc/ld.so.cache\"",
		"load-kmods = true",
		"#no-cgroups = false",
		"#user = \"root:video\"",

		"[nvidia-container-runtime]",
		"#debug = \"/var/log/nvidia-container-runtime.log\"",
		"log-level = \"info\"",
		"mode = \"auto\"",

		"mount-spec-path = \"/etc/nvidia-container-runtime/host-files-for-container.d\"",
	}

	require.Subset(t, lines, expectedLines)
}
