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
	"io/ioutil"
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
	require.NoError(t, ioutil.WriteFile(filename, contents, 0766))

	defer func() { require.NoError(t, os.RemoveAll(testDir)) }()

	cfg, err := GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg.NVIDIAContainerRuntimeConfig.DebugFilePath, "/nvidia-container-toolkit.log")
}

func TestGetConfig(t *testing.T) {
	testCases := []struct {
		description    string
		contents       []string
		expectedError  error
		expectedConfig *Config
	}{
		{
			description: "empty config is default",
			expectedConfig: &Config{
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root: "",
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
					},
				},
				NVIDIACTKConfig: CTKConfig{
					Path: "nvidia-ctk",
				},
			},
		},
		{
			description: "config options set inline",
			contents: []string{
				"nvidia-container-cli.root = \"/bar/baz\"",
				"nvidia-container-runtime.debug = \"/foo/bar\"",
				"nvidia-container-runtime.experimental = true",
				"nvidia-container-runtime.discover-mode = \"not-legacy\"",
				"nvidia-container-runtime.log-level = \"debug\"",
				"nvidia-container-runtime.runtimes = [\"/some/runtime\",]",
				"nvidia-container-runtime.mode = \"not-auto\"",
				"nvidia-container-runtime.modes.csv.mount-spec-path = \"/not/etc/nvidia-container-runtime/host-files-for-container.d\"",
				"nvidia-ctk.path = \"/foo/bar/nvidia-ctk\"",
			},
			expectedConfig: &Config{
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root: "/bar/baz",
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
					},
				},
				NVIDIACTKConfig: CTKConfig{
					Path: "/foo/bar/nvidia-ctk",
				},
			},
		},
		{
			description: "config options set in section",
			contents: []string{
				"[nvidia-container-cli]",
				"root = \"/bar/baz\"",
				"[nvidia-container-runtime]",
				"debug = \"/foo/bar\"",
				"experimental = true",
				"discover-mode = \"not-legacy\"",
				"log-level = \"debug\"",
				"runtimes = [\"/some/runtime\",]",
				"mode = \"not-auto\"",
				"[nvidia-container-runtime.modes.csv]",
				"mount-spec-path = \"/not/etc/nvidia-container-runtime/host-files-for-container.d\"",
				"[nvidia-ctk]",
				"path = \"/foo/bar/nvidia-ctk\"",
			},
			expectedConfig: &Config{
				NVIDIAContainerCLIConfig: ContainerCLIConfig{
					Root: "/bar/baz",
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
					},
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

			cfg, err := loadConfigFrom(reader)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.EqualValues(t, tc.expectedConfig, cfg)
		})
	}
}
