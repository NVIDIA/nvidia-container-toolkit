/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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
	"strings"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestTomlSave(t *testing.T) {
	testCases := []struct {
		description string
		config      *Toml
		expected    string
	}{
		{
			description: "defaultConfig",
			config: func() *Toml {
				t, _ := defaultToml()
				// TODO: We handle the ldconfig path specifically, since this is platform
				// dependent.
				(*toml.Tree)(t).Set("nvidia-container-cli.ldconfig", "OVERRIDDEN")
				return t
			}(),
			expected: `
#accept-nvidia-visible-devices-as-volume-mounts = false
#accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
#swarm-resource = "DOCKER_RESOURCE_GPU"

[nvidia-container-cli]
#debug = "/var/log/nvidia-container-toolkit.log"
environment = []
#ldcache = "/etc/ld.so.cache"
ldconfig = "OVERRIDDEN"
load-kmods = true
#no-cgroups = false
#path = "/usr/bin/nvidia-container-cli"
#root = "/run/nvidia/driver"
#user = "root:video"

[nvidia-container-runtime]
#debug = "/var/log/nvidia-container-runtime.log"
log-level = "info"
mode = "auto"
runtimes = ["docker-runc", "runc"]

[nvidia-container-runtime.modes]

[nvidia-container-runtime.modes.cdi]
annotation-prefixes = ["cdi.k8s.io/"]
default-kind = "nvidia.com/gpu"
spec-dirs = ["/etc/cdi", "/var/run/cdi"]

[nvidia-container-runtime.modes.csv]
mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
path = "nvidia-container-runtime-hook"
skip-mode-detection = false

[nvidia-ctk]
path = "nvidia-ctk"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			buffer := new(bytes.Buffer)
			_, err := tc.config.Save(buffer)
			require.NoError(t, err)

			require.EqualValues(t,
				strings.TrimSpace(tc.expected),
				strings.TrimSpace(buffer.String()),
			)
		})
	}
}

func TestFormat(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "# comment",
			expected: "#comment",
		},
		{
			input:    " #comment",
			expected: "#comment",
		},
		{
			input:    " # comment",
			expected: "#comment",
		},
		{
			input: strings.Join([]string{
				"some",
				"# comment",
				" # comment",
				" #comment",
				"other"}, "\n"),
			expected: strings.Join([]string{
				"some",
				"#comment",
				"#comment",
				"#comment",
				"other"}, "\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, _ := (Toml{}).format([]byte(tc.input))
			require.Equal(t, tc.expected, string(actual))
		})
	}
}

func TestGetFormattedConfig(t *testing.T) {
	expectedLines := []string{
		"#no-cgroups = false",
		"#debug = \"/var/log/nvidia-container-toolkit.log\"",
		"#debug = \"/var/log/nvidia-container-runtime.log\"",
	}

	contents, err := createEmpty().contents()
	require.NoError(t, err)
	lines := strings.Split(string(contents), "\n")

	for _, line := range expectedLines {
		require.Contains(t, lines, line)
	}
}

func TestTomlContents(t *testing.T) {
	testCases := []struct {
		description string
		contents    map[string]interface{}
		expected    string
	}{
		{
			description: "empty config returns commented defaults",
			expected: `
#accept-nvidia-visible-devices-as-volume-mounts = false
#accept-nvidia-visible-devices-envvar-when-unprivileged = true
#swarm-resource = "DOCKER_RESOURCE_GPU"

[nvidia-container-cli]
#debug = "/var/log/nvidia-container-toolkit.log"
#ldcache = "/etc/ld.so.cache"
#no-cgroups = false
#path = "/usr/bin/nvidia-container-cli"
#root = "/run/nvidia/driver"
#user = "root:video"

[nvidia-container-runtime]
#debug = "/var/log/nvidia-container-runtime.log"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tree, err := toml.TreeFromMap(tc.contents)
			require.NoError(t, err)
			cfg := (*Toml)(tree)
			contents, err := cfg.contents()
			require.NoError(t, err)

			require.EqualValues(t,
				strings.TrimSpace(tc.expected),
				strings.TrimSpace(string(contents)),
			)
		})
	}
}

func TestConfigFromToml(t *testing.T) {
	testCases := []struct {
		description    string
		contents       map[string]interface{}
		expectedConfig *Config
	}{
		{
			description: "empty config returns default config",
			contents:    nil,
			expectedConfig: func() *Config {
				c, _ := GetDefault()
				return c
			}(),
		},
		{
			description: "contents overrides default",
			contents: map[string]interface{}{
				"nvidia-container-runtime": map[string]interface{}{
					"debug": "/some/log/file.log",
					"mode":  "csv",
				},
			},
			expectedConfig: func() *Config {
				c, _ := GetDefault()
				c.NVIDIAContainerRuntimeConfig.DebugFilePath = "/some/log/file.log"
				c.NVIDIAContainerRuntimeConfig.Mode = "csv"
				return c
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tomlCfg := fromMap(tc.contents)
			config, err := tomlCfg.Config()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedConfig, config)
		})
	}
}

func fromMap(c map[string]interface{}) *Toml {
	tree, _ := toml.TreeFromMap(c)
	return (*Toml)(tree)
}

func createEmpty() *Toml {
	return fromMap(nil)
}
