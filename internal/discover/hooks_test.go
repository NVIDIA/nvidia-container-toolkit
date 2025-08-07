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

package discover

import (
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/pkg/cdi"
)

func TestCDIHookCreator_isDisabled(t *testing.T) {
	tests := []struct {
		name           string
		disabledHooks  HookName
		enabledHooks   []HookName
		hookName       HookName
		args           []string
		expectedResult *Hook
	}{
		{
			name:           "hook explicitly disabled",
			disabledHooks:  ChmodHook,
			hookName:       ChmodHook,
			args:           []string{"/path/to/file"},
			expectedResult: nil, // ChmodHook is disabled by default and explicitly disabled
		},
		{
			name:          "hook explicitly enabled overrides disabled",
			disabledHooks: ChmodHook,
			enabledHooks:  []HookName{ChmodHook},
			hookName:      ChmodHook,
			args:          []string{"/path/to/file"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "chmod", "--mode", "755", "--path", "/path/to/file"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "hook not in disabled map and not AllHooks disabled",
			disabledHooks: ChmodHook,
			hookName:      UpdateLDCacheHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:           "AllHooks disabled - all hooks should be disabled",
			disabledHooks:  AllHooks,
			hookName:       UpdateLDCacheHook,
			args:           []string{},
			expectedResult: nil,
		},
		{
			name:          "AllHooks disabled but hook explicitly enabled",
			disabledHooks: AllHooks,
			enabledHooks:  []HookName{UpdateLDCacheHook},
			hookName:      UpdateLDCacheHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:           "CreateSymlinksHook requires args - no args provided",
			disabledHooks:  "",
			hookName:       CreateSymlinksHook,
			args:           []string{},
			expectedResult: nil,
		},
		{
			name:          "CreateSymlinksHook requires args - args provided",
			disabledHooks: "",
			hookName:      CreateSymlinksHook,
			args:          []string{"/path/to/symlink"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "/path/to/symlink"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:           "ChmodHook requires args - no args provided",
			disabledHooks:  "",
			hookName:       ChmodHook,
			args:           []string{},
			expectedResult: nil,
		},
		{
			name:          "ChmodHook requires args - args provided",
			disabledHooks: "",
			enabledHooks:  []HookName{ChmodHook}, // Enable ChmodHook since it's disabled by default
			hookName:      ChmodHook,
			args:          []string{"/path/to/file"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "chmod", "--mode", "755", "--path", "/path/to/file"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "UpdateLDCacheHook doesn't require args - no args provided",
			disabledHooks: "",
			hookName:      UpdateLDCacheHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "UpdateLDCacheHook doesn't require args - args provided",
			disabledHooks: "",
			hookName:      UpdateLDCacheHook,
			args:          []string{"arg1", "arg2"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache", "arg1", "arg2"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "EnableCudaCompatHook doesn't require args - no args provided",
			disabledHooks: "",
			hookName:      EnableCudaCompatHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "enable-cuda-compat"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "DisableDeviceNodeModificationHook doesn't require args - no args provided",
			disabledHooks: "",
			hookName:      DisableDeviceNodeModificationHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "disable-device-node-modification"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		// Edge cases
		{
			name:          "empty disabled hooks map",
			disabledHooks: "",
			hookName:      UpdateLDCacheHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "nil disabled hooks map",
			disabledHooks: "",
			hookName:      UpdateLDCacheHook,
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "unknown hook name",
			disabledHooks: ChmodHook,
			hookName:      HookName("unknown-hook"),
			args:          []string{},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "unknown-hook"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:           "AllHooks with unknown hook",
			disabledHooks:  AllHooks,
			hookName:       HookName("unknown-hook"),
			args:           []string{},
			expectedResult: nil,
		},
		// Multiple args tests
		{
			name:          "CreateSymlinksHook with multiple args",
			disabledHooks: "",
			hookName:      CreateSymlinksHook,
			args:          []string{"/path1", "/path2", "/path3"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "/path1", "--link", "/path2", "--link", "/path3"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "ChmodHook with multiple args",
			disabledHooks: "",
			enabledHooks:  []HookName{ChmodHook}, // Enable ChmodHook since it's disabled by default
			hookName:      ChmodHook,
			args:          []string{"/path1", "/path2"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "chmod", "--mode", "755", "--path", "/path1", "--path", "/path2"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:          "UpdateLDCacheHook with multiple args",
			disabledHooks: "",
			hookName:      UpdateLDCacheHook,
			args:          []string{"arg1", "arg2", "arg3"},
			expectedResult: &Hook{
				Lifecycle: cdi.CreateContainerHook,
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "update-ldcache", "arg1", "arg2", "arg3"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.disabledHooks != "" {
				opts = append(opts, WithDisabledHooks(tt.disabledHooks))
			}
			if len(tt.enabledHooks) > 0 {
				opts = append(opts, WithEnabledHooks(tt.enabledHooks...))
			}

			testCreator := NewHookCreator(opts...)

			result := testCreator.Create(tt.hookName, tt.args...)

			require.Equal(t, tt.expectedResult, result)
		})
	}
}
