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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestGetConfigWithCustomConfig(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// By default debug is disabled
	contents := []byte("[nvidia-container-runtime]\ndebug = \"/nvidia-container-toolkit.log\"")
	testDir := path.Join(wd, "test")
	filename := path.Join(testDir, configFileRelativePath)

	previousConfig, present := os.LookupEnv(configOverride)
	os.Setenv(configOverride, testDir)
	defer func() {
		if present {
			os.Setenv(configOverride, previousConfig)
		} else {
			os.Unsetenv(configOverride)
		}
	}()

	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0766))
	require.NoError(t, ioutil.WriteFile(filename, contents, 0766))

	defer func() { require.NoError(t, os.RemoveAll(testDir)) }()

	logger, _ := testlog.NewNullLogger()
	cfg, err := GetConfig(logger)
	require.NoError(t, err)
	require.Equal(t, "/nvidia-container-toolkit.log", cfg.DebugFilePath)
}
