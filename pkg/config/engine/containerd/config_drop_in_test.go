/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

func TestRemoveRuntimePreservesSharedImports(t *testing.T) {
	testCases := map[string]struct {
		additionalFiles    map[string]string
		hostPath           string
		additionalImport   string
		additionalSettings string
		preserveTopLevel   bool
	}{
		"only NVIDIA drop-in": {},
		"another drop-in": {
			additionalFiles:  map[string]string{"10-custom.toml": "[debug]\nlevel = \"debug\"\n"},
			preserveTopLevel: true,
		},
		"nonmatching file": {
			additionalFiles: map[string]string{"notes.txt": "not a drop-in\n"},
		},
		"only NVIDIA drop-in with host path mapping": {
			hostPath: "/host/containerd/conf.d",
		},
		"another drop-in with host path mapping": {
			additionalFiles:  map[string]string{"10-custom.toml": "[debug]\nlevel = \"debug\"\n"},
			hostPath:         "/host/containerd/conf.d",
			preserveTopLevel: true,
		},
		"another import": {
			additionalImport: "/other/containerd/*.toml",
			preserveTopLevel: true,
		},
		"other top-level settings": {
			additionalSettings: "\n[debug]\nlevel = \"debug\"\n",
			preserveTopLevel:   true,
		},
	}

	for _, version := range []int{2, 3} {
		for name, tc := range testCases {
			t.Run(fmt.Sprintf("v%d/%s", version, name), func(t *testing.T) {
				root := t.TempDir()
				dropInDir := filepath.Join(root, "conf.d")
				require.NoError(t, os.Mkdir(dropInDir, 0755))
				dropInPath := filepath.Join(dropInDir, "99-nvidia.toml")
				configPath := filepath.Join(root, "config.toml")
				importDir := dropInDir
				var options []Option
				if tc.hostPath != "" {
					importDir = tc.hostPath
					options = append(options, WithContainerPathAsHostPath(dropInDir, tc.hostPath))
				}
				imports := []string{importDir + "/*.toml"}
				if tc.additionalImport != "" {
					imports = append(imports, tc.additionalImport)
				}
				configBytes, err := toml.Marshal(map[string]any{"version": version, "imports": imports})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(configPath, append(configBytes, tc.additionalSettings...), 0600))
				pluginName := "io.containerd.grpc.v1.cri"
				if version == 3 {
					pluginName = "io.containerd.cri.v1.runtime"
				}
				require.NoError(t, os.WriteFile(dropInPath, fmt.Appendf(nil, "[plugins.%q.containerd.runtimes.nvidia]\nruntime_type = %q\n", pluginName, "io.containerd.runc.v2"), 0600))
				for filename, contents := range tc.additionalFiles {
					require.NoError(t, os.WriteFile(filepath.Join(dropInDir, filename), []byte(contents), 0600))
				}
				options = append(options, WithTopLevelConfigPath(configPath))
				cfg, err := New(options...)
				require.NoError(t, err)
				require.NoError(t, cfg.RemoveRuntime("nvidia"))
				written, err := cfg.Save(dropInPath)
				require.NoError(t, err)
				require.Zero(t, written)
				require.NoFileExists(t, dropInPath)
				for filename, contents := range tc.additionalFiles {
					actual, err := os.ReadFile(filepath.Join(dropInDir, filename))
					require.NoError(t, err)
					require.Equal(t, []byte(contents), actual)
				}
				if !tc.preserveTopLevel {
					require.NoFileExists(t, configPath)
					return
				}
				require.FileExists(t, configPath)
				saved, err := toml.FromFile(configPath).Load()
				require.NoError(t, err)
				actualImports, err := (&Config{Tree: saved}).getStringArrayValue([]string{"imports"})
				require.NoError(t, err)
				require.Equal(t, imports, actualImports)
				require.EqualValues(t, version, saved.Get("version"))
				if tc.additionalSettings != "" {
					require.Equal(t, "debug", saved.GetPath([]string{"debug", "level"}))
				}
			})
		}
	}
}

func TestSimplifyPreservesImportsWithoutDirectoryContents(t *testing.T) {
	testCases := map[string]struct {
		dropInPath string
		imports    []string
	}{
		"stdout": {
			dropInPath: engine.SaveToSTDOUT,
			imports:    []string{"/etc/containerd/conf.d/*.toml"},
		},
		"unavailable directory": {
			dropInPath: filepath.Join(t.TempDir(), "missing", "99-nvidia.toml"),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imports := tc.imports
			if imports == nil {
				imports = []string{filepath.Dir(tc.dropInPath) + "/*.toml"}
			}
			tree, err := toml.FromString(fmt.Sprintf("version = 3\nimports = [%q]\n", imports[0])).Load()
			require.NoError(t, err)
			cfg := topLevelConfig{config: &Config{Tree: tree}, logger: &logger.NullLogger{}}
			cfg.simplify(tc.dropInPath)
			require.EqualValues(t, 3, tree.Get("version"))
			actualImports, err := cfg.config.getStringArrayValue([]string{"imports"})
			require.NoError(t, err)
			require.Equal(t, imports, actualImports)
		})
	}
}

func TestEnsureImports(t *testing.T) {
	testCases := []struct {
		description     string
		configMap       map[string]any
		path            string
		expectedImports []string
	}{
		{
			description:     "empty",
			path:            "/another/path/file.toml",
			expectedImports: []string{"/another/path/*.toml"},
		},
		{
			description: "existing imports as string slice",
			configMap: map[string]any{
				"imports": []string{"/foo/bar/*.toml"},
			},
			path:            "/another/path/file.toml",
			expectedImports: []string{"/foo/bar/*.toml", "/another/path/*.toml"},
		},
		{
			description: "existing imports as any slice",
			configMap: map[string]any{
				"imports": []any{"/foo/bar/*.toml"},
			},
			path:            "/another/path/file.toml",
			expectedImports: []string{"/foo/bar/*.toml", "/another/path/*.toml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cut := topLevelConfig{
				config: &Config{
					Tree: func() *toml.Tree {
						t, _ := toml.FromMap(tc.configMap).Load()
						return t
					}(),
				},
			}

			cut.ensureImports("/another/path/file.toml")
			require.EqualValues(t, tc.expectedImports, cut.config.Get("imports"))
		})
	}

}
