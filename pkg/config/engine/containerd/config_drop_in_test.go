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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

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
