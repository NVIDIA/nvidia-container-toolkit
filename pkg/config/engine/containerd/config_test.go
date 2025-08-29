/**
# Copyright 2024 NVIDIA CORPORATION
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

package containerd

import (
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

func TestAddRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description    string
		config         string
		setAsDefault   bool
		expectedConfig string
		expectedError  error
	}{
		{
			description: "empty config not default runtime",
			config:      `version = 1`,
			expectedConfig: `
							version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = ""
					runtime_root = ""
					runtime_type = "io.containerd.runc.v2"
					[plugins."cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
			`,
			expectedError: nil,
		},
		{
			description: "options from runc are imported",
			config: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."cri".containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from default runtime are imported",
			config: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				default_runtime_name = "default"
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				default_runtime_name = "default"
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.default]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = true
					[plugins."cri".containerd.runtimes.test]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = true
				`,
		},
		{
			description: "options from the default runtime take precedence over runc",
			config: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				default_runtime_name = "default"
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."cri".containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
			`,
			expectedConfig: `
			version = 1
			[plugins]
			[plugins."cri"]
				[plugins."cri".containerd]
				default_runtime_name = "default"
				[plugins."cri".containerd.runtimes]
					[plugins."cri".containerd.runtimes.runc]
					privileged_without_host_devices = true
					runtime_engine = "engine"
					runtime_root = "root"
					runtime_type = "type"
					[plugins."cri".containerd.runtimes.runc.options]
						BinaryName = "/usr/bin/runc"
						SystemdCgroup = true
					[plugins."cri".containerd.runtimes.default]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."cri".containerd.runtimes.default.options]
						BinaryName = "/usr/bin/default"
						SystemdCgroup = false
					[plugins."cri".containerd.runtimes.test]
					privileged_without_host_devices = false
					runtime_engine = "defaultengine"
					runtime_root = "defaultroot"
					runtime_type = "defaulttype"
					[plugins."cri".containerd.runtimes.test.options]
						BinaryName = "/usr/bin/test"
						Runtime = "/usr/bin/test"
						SystemdCgroup = false
				`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			expectedConfig, err := toml.Load(tc.expectedConfig)
			require.NoError(t, err)

			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(tc.config)),
			)
			require.NoError(t, err)

			err = c.AddRuntime("test", "/usr/bin/test", tc.setAsDefault)
			require.NoError(t, err)

			require.EqualValues(t, expectedConfig.String(), c.String())
		})
	}
}

func TestGetRuntimeConfig(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	config := `
	version = 2
	[plugins]
	[plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      ignore_blockio_not_enabled_errors = false
      ignore_rdt_not_enabled_errors = false
      no_pivot = false
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        privileged_without_host_devices_all_devices_allowed = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""
        sandbox_mode = ""
        snapshotter = ""

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          base_runtime_spec = ""
          cni_conf_dir = ""
          cni_max_conf_num = 0
          container_annotations = []
          pod_annotations = []
          privileged_without_host_devices = false
          privileged_without_host_devices_all_devices_allowed = false
          runtime_engine = ""
          runtime_path = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"
          sandbox_mode = "podsandbox"
          snapshotter = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            CriuImagePath = ""
            CriuPath = ""
            CriuWorkPath = ""
            IoGid = 0
            IoUid = 0
            NoNewKeyring = false
            NoPivotRoot = false
            Root = ""
            ShimCgroup = ""
            SystemdCgroup = false
`
	testCases := []struct {
		description   string
		runtime       string
		expected      string
		expectedError error
	}{
		{
			description:   "valid runtime config, existing runtime",
			runtime:       "runc",
			expected:      "/usr/bin/runc",
			expectedError: nil,
		},
		{
			description:   "valid runtime config, non-existing runtime",
			runtime:       "some-other-runtime",
			expected:      "",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(config)),
			)
			require.NoError(t, err)

			rc, err := c.GetRuntimeConfig(tc.runtime)
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expected, rc.GetBinaryPath())
		})
	}
}

func TestNew_Version2Config_UsesDropIn(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a version 2 config
	config, err := toml.TreeFromMap(map[string]interface{}{
		"version": int64(2),
	})
	require.NoError(t, err)
	_, err = config.Save(configPath)
	require.NoError(t, err)

	// Create config - drop-in is now automatic for v2
	cfg, err := New(
		WithLogger(logger.New()),
		WithPath(configPath),
		WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should return a Config for version 2 (with drop-in support)
	configCfg, ok := cfg.(*Config)
	assert.True(t, ok)
	assert.Equal(t, int64(2), configCfg.Version)
	assert.NotNil(t, configCfg.NVConfig)
}

func TestNew_Version1Config_NoDropIn(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a version 1 config
	config, err := toml.TreeFromMap(map[string]interface{}{
		"version": int64(1),
	})
	require.NoError(t, err)
	_, err = config.Save(configPath)
	require.NoError(t, err)

	// Create config with drop-in support
	cfg, err := New(
		WithLogger(logger.New()),
		WithPath(configPath),
		WithConfigSource(toml.FromFile(configPath)),
		WithUseLegacyConfig(true),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should return a ConfigV1 for version 1 (no drop-in)
	_, ok := cfg.(*ConfigV1)
	assert.True(t, ok)
}

func TestNew_NoConfig_CreatesDropIn(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create config (no existing file) - defaults to v2 with drop-in
	cfg, err := New(
		WithLogger(logger.New()),
		WithPath(configPath),
		WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should return a Config for new configs (defaults to v2)
	configCfg, ok := cfg.(*Config)
	assert.True(t, ok)
	assert.Equal(t, int64(2), configCfg.Version)
	assert.NotNil(t, configCfg.NVConfig)
}

func TestNew_Version3Config_UsesDropIn(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a version 3 config
	config, err := toml.TreeFromMap(map[string]interface{}{
		"version": int64(3),
	})
	require.NoError(t, err)
	_, err = config.Save(configPath)
	require.NoError(t, err)

	// Create config - drop-in is automatic for v3 (any non-v1)
	cfg, err := New(
		WithLogger(logger.New()),
		WithPath(configPath),
		WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should return a Config for version 3 (with drop-in support)
	configCfg, ok := cfg.(*Config)
	assert.True(t, ok)
	assert.Equal(t, int64(3), configCfg.Version)
	assert.NotNil(t, configCfg.NVConfig)
}

func TestAddRuntimeWithDropIn(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	t.Run("v2 config creates drop-in configuration", func(t *testing.T) {
		config := `version = 2`
		c, err := New(
			WithLogger(logger),
			WithConfigSource(toml.FromString(config)),
		)
		require.NoError(t, err)

		// Should be a Config for v2
		configCfg, ok := c.(*Config)
		require.True(t, ok)

		// AddRuntime should modify NVConfig tree for drop-in
		err = configCfg.AddRuntime("test", "/usr/bin/test", true)
		require.NoError(t, err)

		// Verify NVConfig has the runtime configuration
		runtimePath := configCfg.NVConfig.GetPath([]string{"plugins", configCfg.CRIRuntimePluginName, "containerd", "runtimes", "test", "options", "BinaryName"})
		assert.Equal(t, "/usr/bin/test", runtimePath)

		// Verify default runtime is set
		defaultRuntime := configCfg.NVConfig.GetPath([]string{"plugins", configCfg.CRIRuntimePluginName, "containerd", "default_runtime_name"})
		assert.Equal(t, "test", defaultRuntime)

		// Main tree should remain unchanged
		assert.Equal(t, "version = 2\n", configCfg.Tree.String())
	})

	t.Run("v3 config creates drop-in configuration", func(t *testing.T) {
		config := `version = 3`
		c, err := New(
			WithLogger(logger),
			WithConfigSource(toml.FromString(config)),
		)
		require.NoError(t, err)

		// Should be a Config for v3
		configCfg, ok := c.(*Config)
		require.True(t, ok)

		// AddRuntime should modify NVConfig tree for drop-in
		err = configCfg.AddRuntime("nvidia", "/usr/bin/nvidia-container-runtime", false)
		require.NoError(t, err)

		// Verify NVConfig has the runtime configuration
		runtimePath := configCfg.NVConfig.GetPath([]string{"plugins", configCfg.CRIRuntimePluginName, "containerd", "runtimes", "nvidia", "options", "BinaryName"})
		assert.Equal(t, "/usr/bin/nvidia-container-runtime", runtimePath)

		// Default runtime should not be set
		defaultRuntime := configCfg.NVConfig.GetPath([]string{"plugins", configCfg.CRIRuntimePluginName, "containerd", "default_runtime_name"})
		assert.Nil(t, defaultRuntime)

		// Main tree should remain unchanged
		assert.Equal(t, "version = 3\n", configCfg.Tree.String())
	})
}

func TestConfig_EnableCDI(t *testing.T) {
	// Test that Config stores CDI configuration in NVConfig
	t.Run("v2 config stores CDI in drop-in", func(t *testing.T) {
		tree, err := toml.TreeFromMap(map[string]interface{}{"version": int64(2)})
		require.NoError(t, err)
		nvConfig, err := toml.TreeFromMap(map[string]interface{}{})
		require.NoError(t, err)
		cfg := &Config{
			Tree:                 tree,
			NVConfig:             nvConfig,
			Logger:               logger.New(),
			Version:              2,
			CRIRuntimePluginName: "io.containerd.grpc.v1.cri",
		}

		cfg.EnableCDI()

		// Should have enabled CDI in NVConfig
		cdiEnabled := cfg.NVConfig.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
		assert.Equal(t, true, cdiEnabled)

		// Main tree should not be modified
		cdiInMain := cfg.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
		assert.Nil(t, cdiInMain)
	})
}

func TestDropInIntegration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	dropInDir := filepath.Join(tempDir, "conf.d")
	dropInPath := filepath.Join(dropInDir, "99.nvidia.toml")

	// Create minimal config with version 2
	config, err := toml.TreeFromMap(map[string]interface{}{
		"version": int64(2),
	})
	require.NoError(t, err)
	_, err = config.Save(configPath)
	require.NoError(t, err)

	// Create config with drop-in support
	cfg, err := New(
		WithLogger(logger.New()),
		WithPath(configPath),
		WithDropInDir(dropInDir),
		WithConfigSource(toml.FromFile(configPath)),
	)
	require.NoError(t, err)

	// Add runtime
	err = cfg.AddRuntime("nvidia", "/usr/bin/nvidia-container-runtime", true)
	require.NoError(t, err)

	// Enable CDI
	cfg.EnableCDI()

	// Save drop-in configuration
	_, err = cfg.Save(dropInPath)
	require.NoError(t, err)

	// Verify drop-in was created
	_, err = os.Stat(dropInPath)
	require.NoError(t, err)

	// Verify drop-in content
	dropInTree, err := toml.LoadFile(dropInPath)
	require.NoError(t, err)

	// Check runtime configuration
	runtimeType := dropInTree.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia", "runtime_type"})
	assert.Equal(t, "io.containerd.runc.v2", runtimeType)

	// Check default runtime
	defaultRuntime := dropInTree.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"})
	assert.Equal(t, "nvidia", defaultRuntime)

	// Check CDI enabled
	cdiEnabled := dropInTree.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "enable_cdi"})
	assert.Equal(t, true, cdiEnabled)

	// Test removal
	err = cfg.RemoveRuntime("nvidia")
	require.NoError(t, err)

	// Verify drop-in is removed
	_, err = os.Stat(dropInPath)
	assert.True(t, os.IsNotExist(err))
}
