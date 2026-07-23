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

func TestNewApplicationProfileHookDiscoverer(t *testing.T) {
	t.Run("enabled hook creator returns a single hook", func(t *testing.T) {
		hookCreator := NewHookCreator(WithNVIDIACDIHookPath(defaultNvidiaCDIHookPath))

		d := NewApplicationProfileHookDiscoverer(hookCreator)

		hooks, err := d.Hooks()
		require.NoError(t, err)
		require.Len(t, hooks, 1)
		require.Equal(t, "createContainer", hooks[0].Lifecycle)
		require.Equal(t, defaultNvidiaCDIHookPath, hooks[0].Path)
		require.Equal(t, []string{"nvidia-cdi-hook", "update-application-profile"}, hooks[0].Args)

		devices, err := d.Devices()
		require.NoError(t, err)
		require.Empty(t, devices)

		mounts, err := d.Mounts()
		require.NoError(t, err)
		require.Empty(t, mounts)
	})

	t.Run("all-disabled hook creator returns no hook", func(t *testing.T) {
		hookCreator := &allDisabledHookCreator{}

		d := NewApplicationProfileHookDiscoverer(hookCreator)

		hooks, err := d.Hooks()
		require.NoError(t, err)
		require.Empty(t, hooks)
	})

	t.Run("explicitly disabled hook returns no hook", func(t *testing.T) {
		hookCreator := NewHookCreator(WithDisabledHooks(ApplicationProfileHook))

		d := NewApplicationProfileHookDiscoverer(hookCreator)

		hooks, err := d.Hooks()
		require.NoError(t, err)
		require.Empty(t, hooks)
	})
}
