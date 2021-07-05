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
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestComposite(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	// Empty list
	c := newListWithLogger(logger)

	cfg := &Config{}
	err := c.Update(cfg)

	require.NoError(t, err)
	require.EqualValues(t, &Config{}, cfg)

	// Add a single mock config
	mockConfigUpdater := &configUpdaterMock{
		UpdateFunc: func(cfg *Config) error {
			cfg.DebugFilePath = "updated"
			return nil
		},
	}

	c = newListWithLogger(logger, mockConfigUpdater)
	cfg = &Config{}
	err = c.Update(cfg)
	require.NoError(t, err)
	require.EqualValues(t, &Config{DebugFilePath: "updated"}, cfg)

	require.Len(t, mockConfigUpdater.UpdateCalls(), 1)

	// Reset the calls
	mockConfigUpdater.calls = struct{ Update []struct{ Config *Config } }{}

	// define an error config
	errorConfigUpdater := &configUpdaterMock{
		UpdateFunc: func(cfg *Config) error {
			return fmt.Errorf("mock error")
		},
	}

	c = newListWithLogger(logger, errorConfigUpdater, mockConfigUpdater)
	cfg = &Config{}
	err = c.Update(cfg)
	require.Error(t, err)
	require.EqualValues(t, &Config{}, cfg)

	require.Len(t, errorConfigUpdater.UpdateCalls(), 1)
	require.Len(t, mockConfigUpdater.UpdateCalls(), 0)

	// Reset the calls
	mockConfigUpdater.calls = struct{ Update []struct{ Config *Config } }{}
	errorConfigUpdater.calls = struct{ Update []struct{ Config *Config } }{}

	// Change the order of the config and error
	c = newListWithLogger(logger, mockConfigUpdater, errorConfigUpdater)
	cfg = &Config{}
	err = c.Update(cfg)
	require.Error(t, err)
	require.EqualValues(t, &Config{DebugFilePath: "updated"}, cfg)

	require.Len(t, errorConfigUpdater.UpdateCalls(), 1)
	require.Len(t, mockConfigUpdater.UpdateCalls(), 1)

	// Reset the calls
	mockConfigUpdater.calls = struct{ Update []struct{ Config *Config } }{}
	errorConfigUpdater.calls = struct{ Update []struct{ Config *Config } }{}

	// Call the mock twice
	c = newListWithLogger(logger, mockConfigUpdater, mockConfigUpdater)
	cfg = &Config{}
	err = c.Update(cfg)
	require.NoError(t, err)
	require.EqualValues(t, &Config{DebugFilePath: "updated"}, cfg)

	require.Len(t, mockConfigUpdater.UpdateCalls(), 2)
}
