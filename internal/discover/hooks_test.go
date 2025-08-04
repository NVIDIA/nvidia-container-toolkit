/**
# Copyright 2025 NVIDIA CORPORATION
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

	"github.com/stretchr/testify/assert"
)

func TestChmodHookDisabledByDefault(t *testing.T) {
	testCases := []struct {
		name       string
		opts       []Option
		expectHook bool
	}{
		{
			name:       "chmod hook disabled by default",
			opts:       []Option{},
			expectHook: false,
		},
		{
			name:       "chmod hook disabled when explicitly disabled",
			opts:       []Option{WithEnableChmodHook(false)},
			expectHook: false,
		},
		{
			name:       "chmod hook enabled when explicitly enabled",
			opts:       []Option{WithEnableChmodHook(true)},
			expectHook: true,
		},
		{
			name: "chmod hook disabled via DisabledHooks even when enabled",
			opts: []Option{
				WithEnableChmodHook(true),
				WithDisabledHooks(ChmodHook),
			},
			expectHook: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creator := NewHookCreator(tc.opts...)
			hook := creator.Create(ChmodHook, "/dev/dri")

			if tc.expectHook {
				assert.NotNil(t, hook, "Expected chmod hook to be created")
			} else {
				assert.Nil(t, hook, "Expected chmod hook to be nil")
			}
		})
	}
}

func TestChmodHookRequiresArgs(t *testing.T) {
	// Even when enabled, chmod hook should be disabled if no args are provided
	creator := NewHookCreator(WithEnableChmodHook(true))
	hook := creator.Create(ChmodHook)
	assert.Nil(t, hook, "Expected chmod hook to be nil when no args provided")
}

func TestOtherHooksNotAffected(t *testing.T) {
	// Ensure other hooks are not affected by the chmod hook changes
	creator := NewHookCreator()

	// CreateSymlinksHook should still work when args are provided
	symlinkHook := creator.Create(CreateSymlinksHook, "/source::/target")
	assert.NotNil(t, symlinkHook, "Expected create symlinks hook to be created")

	// CreateSymlinksHook should be nil when no args
	symlinkHookNoArgs := creator.Create(CreateSymlinksHook)
	assert.Nil(t, symlinkHookNoArgs, "Expected create symlinks hook to be nil when no args")
}

func TestAllHooksDisabled(t *testing.T) {
	creator := NewHookCreator(
		WithEnableChmodHook(true),
		WithDisabledHooks(AllHooks),
	)

	// When AllHooks is disabled, the creator should return an allDisabledHookCreator
	// which returns nil for all hooks
	hook := creator.Create(ChmodHook, "/dev/dri")
	assert.Nil(t, hook, "Expected no hooks when AllHooks is disabled")
}
