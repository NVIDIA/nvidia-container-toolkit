/*
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

package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestUpdateFromReader(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		readerError   bool
		lines         []string
		expected      *Config
		expectedError bool
	}{
		{
			description:   "reader error returns error",
			readerError:   true,
			expectedError: true,
			expected:      getDefaultConfig(),
		},
		{
			description: "empty config returns defaults",
			lines:       []string{},
			expected: &Config{
				DebugFilePath: "/dev/null",
				LogLevel:      "info",
			},
		},
		{
			description: "debug output is set",
			lines:       []string{"nvidia-container-runtime.debug=\"nvidia-container-toolkit.log\""},
			expected: &Config{
				DebugFilePath: "nvidia-container-toolkit.log",
				LogLevel:      "info",
			},
		},
		{
			description: "debug output is set in section",
			lines:       []string{"[nvidia-container-runtime]", "debug=\"nvidia-container-toolkit.log\""},
			expected: &Config{
				DebugFilePath: "nvidia-container-toolkit.log",
				LogLevel:      "info",
			},
		},
		{
			description: "blank debug is set",
			lines:       []string{"nvidia-container-runtime.debug=\"\""},
			expected: &Config{
				DebugFilePath: "",
				LogLevel:      "info",
			},
		},
		{
			description: "non-string debug is ignored",
			lines:       []string{"nvidia-container-runtime.debug=2"},
			expected: &Config{
				DebugFilePath: "/dev/null",
				LogLevel:      "info",
			},
		},
	}

	for i, tc := range testCases {
		cfg := getDefaultConfig()

		c := tomlConfig{
			logger: logger,
			sections: []tomlSection{
				{section: nvidiaContainerRuntimeConfigSection},
			},
		}
		var reader io.Reader
		if tc.readerError {
			reader = iotest.ErrReader(fmt.Errorf("error"))
		} else {
			reader = strings.NewReader(strings.Join(tc.lines, "\n"))
		}

		err := c.updateFromReader(cfg, reader)

		if tc.expectedError {
			require.Error(t, err, "%d: %v", i, tc.description)
		} else {
			require.NoError(t, err, "%d: %v", i, tc.description)
		}
		require.EqualValues(t, tc.expected, cfg, "%d: %v", i, tc.description)
	}
}

func TestUpdateFromReaderExperimental(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		readerError   bool
		lines         []string
		expected      *Config
		expectedError bool
	}{
		{
			lines:    []string{"nvidia-container-runtime.debug=\"nvidia-container-toolkit.log\""},
			expected: getDefaultConfig(),
		},
		{
			lines:    []string{"[nvidia-container-runtime]", "debug=\"nvidia-container-toolkit.log\""},
			expected: getDefaultConfig(),
		},
		{
			lines: []string{"nvidia-container-runtime.experimental.debug=\"\""},
			expected: &Config{
				DebugFilePath: "",
				LogLevel:      "info",
			},
		},
		{
			lines:    []string{"nvidia-container-runtime.experimental.debug=2"},
			expected: getDefaultConfig(),
		},
		{
			lines: []string{"nvidia-container-runtime.experimental.debug=\"nvidia-container-toolkit.log\""},
			expected: &Config{
				DebugFilePath: "nvidia-container-toolkit.log",
				LogLevel:      "info",
			},
		},
		{
			lines: []string{"[nvidia-container-runtime.experimental]", "debug=\"nvidia-container-toolkit.log\""},
			expected: &Config{
				DebugFilePath: "nvidia-container-toolkit.log",
				LogLevel:      "info",
			},
		},
		{
			lines: []string{
				"nvidia-container-runtime.debug=\"nvidia-container-toolkit.log\"",
				"nvidia-container-runtime.experimental.debug=\"nvidia-container-exp-toolkit.log\"",
			},
			expected: &Config{
				DebugFilePath: "nvidia-container-exp-toolkit.log",
				LogLevel:      "info",
			},
		},
	}

	for i, tc := range testCases {
		cfg := getDefaultConfig()

		c := tomlConfig{
			logger: logger,
			sections: []tomlSection{
				{section: nvidiaContainerRuntimeExperimentalConfigSection},
			},
		}
		var reader io.Reader
		if tc.readerError {
			reader = iotest.ErrReader(fmt.Errorf("error"))
		} else {
			reader = strings.NewReader(strings.Join(tc.lines, "\n"))
		}

		err := c.updateFromReader(cfg, reader)

		if tc.expectedError {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}
		require.EqualValues(t, tc.expected, cfg, "%d: %v", i, tc)
	}
}

func TestConfigFromFile(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// By default debug is disabled
	lines := []string{
		"[nvidia-container-cli]",
		"root = \"/run/nvidia/driver\"",
		"[nvidia-container-runtime]",
		"#debug = \"/nvidia-container-toolkit.log\"",
		"",
		"[nvidia-container-runtime.experimental]",
		"debug = \"/nvidia-container-toolkit.experimental.log\"",
	}

	contents := []byte(strings.Join(lines, "\n"))
	testDir := path.Join(wd, "test")
	filename := path.Join(testDir, configFileRelativePath)

	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0766))
	require.NoError(t, ioutil.WriteFile(filename, contents, 0766))
	defer func() { require.NoError(t, os.RemoveAll(testDir)) }()

	logger, _ := testlog.NewNullLogger()
	c := newConfigFromFileWithLogger(logger, filename)

	cfg := getDefaultConfig()

	err = c.Update(cfg)

	require.NoError(t, err)
	require.Equal(t, "/nvidia-container-toolkit.experimental.log", cfg.DebugFilePath)

	require.Equal(t, "/run/nvidia/driver", cfg.Root)
}

func TestConfigFromNonexistentFileReturnsNoop(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	c := newConfigFromFileWithLogger(logger, "/does/not/exist")

	n, ok := c.(*noop)

	require.True(t, ok)
	require.NotNil(t, n)
}

func TestGetConfigKey(t *testing.T) {
	require.Equal(t, "key", tomlSection{}.configKey("key"))
	require.Equal(t, "section.key", tomlSection{section: "section"}.configKey("key"))
}
