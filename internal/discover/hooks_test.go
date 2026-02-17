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
)

func TestNewHookCreator(t *testing.T) {
	testCases := []struct {
		name     string
		opts     []Option
		expected HookCreator
	}{
		{
			name: "default creates cdiHookCreator with default path",
			opts: nil,
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					ChmodHook: true, // ChmodHook is disabled by default
				},
			},
		},
		{
			name: "custom nvidia-cdi-hook path",
			opts: []Option{
				WithNVIDIACDIHookPath("/custom/path/nvidia-cdi-hook"),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: "/custom/path/nvidia-cdi-hook",
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					ChmodHook: true,
				},
			},
		},
		{
			name: "AllHooks disabled returns allDisabledHookCreator",
			opts: []Option{
				WithDisabledHooks(AllHooks),
			},
			expected: &allDisabledHookCreator{},
		},
		{
			name: "AllHooks disabled but UpdateLDCacheHook enabled",
			opts: []Option{
				WithDisabledHooks(AllHooks),
				WithEnabledHooks(UpdateLDCacheHook),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					AllHooks:          true,
					UpdateLDCacheHook: false,
					ChmodHook:         true,
				},
			},
		},
		{
			name: "multiple hooks disabled and enabled",
			opts: []Option{
				WithDisabledHooks(UpdateLDCacheHook, CreateSymlinksHook, EnableCudaCompatHook, DisableDeviceNodeModificationHook),
				WithEnabledHooks(ChmodHook, UpdateLDCacheHook),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					UpdateLDCacheHook:                 false,
					CreateSymlinksHook:                true,
					EnableCudaCompatHook:              true,
					ChmodHook:                         false,
					DisableDeviceNodeModificationHook: true,
				},
			},
		},
		{
			name: "WithDisabledHooks can be called multiple times",
			opts: []Option{
				WithDisabledHooks(UpdateLDCacheHook),
				WithDisabledHooks(CreateSymlinksHook),
				WithDisabledHooks(EnableCudaCompatHook),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					UpdateLDCacheHook:    true,
					CreateSymlinksHook:   true,
					EnableCudaCompatHook: true,
					ChmodHook:            true, // Default disabled
				},
			},
		},
		{
			name: "WithEnabledHooks overrides defaultDisabledHooks",
			opts: []Option{
				WithEnabledHooks(ChmodHook),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: defaultNvidiaCDIHookPath,
				fixedArgs:         []string{"nvidia-cdi-hook"},
				disabledHooks: map[HookName]bool{
					ChmodHook: false, // ChmodHook is enabled
				},
			},
		},
		{
			name: "nvidia-ctk binary path affects fixed args",
			opts: []Option{
				WithNVIDIACDIHookPath("/usr/bin/nvidia-ctk"),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: "/usr/bin/nvidia-ctk",
				fixedArgs:         []string{"nvidia-ctk", "hook"},
				disabledHooks: map[HookName]bool{
					ChmodHook: true,
				},
			},
		},
		{
			name: "nvidia-ctk in custom NVIDIA toolkit path",
			opts: []Option{
				WithNVIDIACDIHookPath("/usr/local/nvidia/toolkit/nvidia-ctk"),
			},
			expected: &cdiHookCreator{
				nvidiaCDIHookPath: "/usr/local/nvidia/toolkit/nvidia-ctk",
				fixedArgs:         []string{"nvidia-ctk", "hook"},
				disabledHooks: map[HookName]bool{
					ChmodHook: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hookCreator := NewHookCreator(tc.opts...)

			require.EqualValues(t, tc.expected, hookCreator)
		})
	}
}
func TestCDIHookCreator_Create(t *testing.T) {
	testCases := []struct {
		name         string
		hookCreator  HookCreator
		hookName     HookName
		args         []string
		expectedHook *Hook
	}{
		{
			name:        "CreateSymlinksHook with args",
			hookCreator: NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:    CreateSymlinksHook,
			args:        []string{"/source::/target", "/source2::/target2"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      "/usr/bin/nvidia-cdi-hook",
				Args:      []string{"nvidia-cdi-hook", "create-symlinks", "--link", "/source::/target", "--link", "/source2::/target2"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:         "CreateSymlinksHook without args returns nil",
			hookCreator:  NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:     CreateSymlinksHook,
			args:         []string{},
			expectedHook: nil,
		},
		{
			name: "ChmodHook with args (when enabled)",
			hookCreator: NewHookCreator(
				WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath),
				WithEnabledHooks(ChmodHook),
			),
			hookName: ChmodHook,
			args:     []string{"/path/to/file1", "/path/to/file2"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "chmod", "--mode", "755", "--path", "/path/to/file1", "--path", "/path/to/file2"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:         "ChmodHook disabled by default returns nil",
			hookCreator:  NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:     ChmodHook,
			args:         []string{"/path/to/file"},
			expectedHook: nil, // ChmodHook is disabled by default
		},
		{
			name:        "UpdateLDCacheHook with no args",
			hookCreator: NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:    UpdateLDCacheHook,
			args:        []string{},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:        "UpdateLDCacheHook with args",
			hookCreator: NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:    UpdateLDCacheHook,
			args:        []string{"/usr/lib64"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/usr/lib64"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:        "EnableCudaCompatHook",
			hookCreator: NewHookCreator(),
			hookName:    EnableCudaCompatHook,
			args:        []string{"--root", "/some/root"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "enable-cuda-compat", "--root", "/some/root"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:        "DisableDeviceNodeModificationHook",
			hookCreator: NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath)),
			hookName:    DisableDeviceNodeModificationHook,
			args:        []string{},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "disable-device-node-modification"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:        "nvidia-ctk binary uses different args format",
			hookCreator: NewHookCreator(WithNVIDIACDIHookPath("/usr/bin/nvidia-ctk")),
			hookName:    UpdateLDCacheHook,
			args:        []string{},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      "/usr/bin/nvidia-ctk",
				Args:      []string{"nvidia-ctk", "hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name: "nvidia-ctk in custom NVIDIA toolkit path",
			hookCreator: NewHookCreator(
				WithNVIDIACDIHookPath("/usr/local/nvidia/toolkit/nvidia-ctk"),
			),
			hookName: EnableCudaCompatHook,
			args:     []string{"--root", "/some/root"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      "/usr/local/nvidia/toolkit/nvidia-ctk",
				Args:      []string{"nvidia-ctk", "hook", "enable-cuda-compat", "--root", "/some/root"},
				Env:       []string{"NVIDIA_CTK_DEBUG=false"},
			},
		},
		{
			name:         "hook disabled when in disabledHooks list",
			hookCreator:  NewHookCreator(WithDisabledHooks(UpdateLDCacheHook)),
			hookName:     UpdateLDCacheHook,
			args:         []string{},
			expectedHook: nil,
		},
		{
			name:         "allDisabled with unknown hook",
			hookCreator:  &allDisabledHookCreator{},
			hookName:     HookName("unknown-hook"),
			args:         []string{},
			expectedHook: nil,
		},
		{
			name:         "allDisabled with defined hook",
			hookCreator:  &allDisabledHookCreator{},
			hookName:     UpdateLDCacheHook,
			args:         []string{},
			expectedHook: nil,
		},
		{
			name:        "debug logging enabled",
			hookCreator: NewHookCreator(WithDebugLogging(true)),
			hookName:    UpdateLDCacheHook,
			args:        []string{},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      defaultNvidiaCDIHookPath,
				Args:      []string{"nvidia-cdi-hook", "update-ldcache"},
				Env:       []string{"NVIDIA_CTK_DEBUG=true"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hookCreator := tc.hookCreator

			hook := hookCreator.Create(tc.hookName, tc.args...)
			require.Equal(t, tc.expectedHook, hook)
		})
	}
}

func TestCDIHookCreator_isDisabled(t *testing.T) {
	tests := []struct {
		name           string
		disabledHooks  []HookName
		enabledHooks   []HookName
		hookName       HookName
		args           []string
		expectedResult bool
	}{
		{
			name:           "hook explicitly disabled",
			disabledHooks:  []HookName{ChmodHook},
			hookName:       ChmodHook,
			args:           []string{"/path/to/file"},
			expectedResult: true, // ChmodHook is disabled by default and explicitly disabled
		},
		{
			name:           "hook explicitly enabled overrides disabled",
			disabledHooks:  []HookName{ChmodHook},
			enabledHooks:   []HookName{ChmodHook},
			hookName:       ChmodHook,
			args:           []string{"/path/to/file"},
			expectedResult: false,
		},
		{
			name:           "hook not in disabled map and not AllHooks disabled",
			disabledHooks:  []HookName{ChmodHook},
			hookName:       UpdateLDCacheHook,
			args:           []string{},
			expectedResult: false,
		},
		{
			name:           "CreateSymlinksHook requires args - no args provided",
			disabledHooks:  []HookName{},
			hookName:       CreateSymlinksHook,
			args:           []string{},
			expectedResult: true,
		},
		{
			name:           "CreateSymlinksHook requires args - args provided",
			disabledHooks:  []HookName{},
			hookName:       CreateSymlinksHook,
			args:           []string{"/path/to/symlink"},
			expectedResult: false,
		},
		{
			name:           "ChmodHook requires args - no args provided",
			disabledHooks:  []HookName{},
			hookName:       ChmodHook,
			args:           []string{},
			expectedResult: true,
		},
		{
			name:           "ChmodHook requires args - args provided",
			disabledHooks:  []HookName{},
			enabledHooks:   []HookName{ChmodHook}, // Enable ChmodHook since it's disabled by default
			hookName:       ChmodHook,
			args:           []string{"/path/to/file"},
			expectedResult: false,
		},
		{
			name:           "UpdateLDCacheHook doesn't require args - no args provided",
			disabledHooks:  []HookName{},
			hookName:       UpdateLDCacheHook,
			args:           []string{},
			expectedResult: false,
		},
		{
			name:           "UpdateLDCacheHook doesn't require args - args provided",
			disabledHooks:  []HookName{},
			hookName:       UpdateLDCacheHook,
			args:           []string{"arg1", "arg2"},
			expectedResult: false,
		},
		{
			name:           "EnableCudaCompatHook doesn't require args - no args provided",
			disabledHooks:  []HookName{},
			hookName:       EnableCudaCompatHook,
			args:           []string{},
			expectedResult: false,
		},
		{
			name:           "DisableDeviceNodeModificationHook doesn't require args - no args provided",
			disabledHooks:  []HookName{},
			hookName:       DisableDeviceNodeModificationHook,
			args:           []string{},
			expectedResult: false,
		},
		// Edge cases
		{
			name:           "nil disabled hooks map",
			disabledHooks:  nil,
			hookName:       UpdateLDCacheHook,
			args:           []string{},
			expectedResult: false,
		},
		{
			name:           "unknown hook name",
			disabledHooks:  []HookName{ChmodHook},
			hookName:       HookName("unknown-hook"),
			args:           []string{},
			expectedResult: false,
		},
		// Multiple args tests
		{
			name:           "CreateSymlinksHook with multiple args",
			disabledHooks:  []HookName{},
			hookName:       CreateSymlinksHook,
			args:           []string{"/path1", "/path2", "/path3"},
			expectedResult: false,
		},
		{
			name:           "ChmodHook with multiple args",
			disabledHooks:  []HookName{},
			enabledHooks:   []HookName{ChmodHook}, // Enable ChmodHook since it's disabled by default
			hookName:       ChmodHook,
			args:           []string{"/path1", "/path2"},
			expectedResult: false,
		},
		{
			name:           "UpdateLDCacheHook with multiple args",
			disabledHooks:  []HookName{},
			hookName:       UpdateLDCacheHook,
			args:           []string{"arg1", "arg2", "arg3"},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCreator := NewHookCreator(
				WithDisabledHooks(tt.disabledHooks...),
				WithEnabledHooks(tt.enabledHooks...),
			)

			require.Equal(t, testCreator.(*cdiHookCreator).isDisabled(tt.hookName, tt.args...), tt.expectedResult)
		})
	}
}
