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

package nvcdi

import (
	"path/filepath"
	"slices"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestMigCapsValidation(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description string
		ids         []string
		expectedErr string
	}{
		{
			description: "unknown capability is rejected",
			ids:         []string{"bogus"},
			expectedErr: "invalid MIG capability",
		},
		{
			description: "per-instance access cap is rejected",
			ids:         []string{"gpu0/gi0/access"},
			expectedErr: "invalid MIG capability",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			lib, err := New(
				WithLogger(logger),
				WithMode(ModeMigCaps),
			)
			require.NoError(t, err)

			_, err = lib.GetDeviceSpecsByID(tc.ids...)
			require.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestMigCapsMode(t *testing.T) {
	defer devices.SetAllForTest()()

	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)
	hostRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-mig")

	lib, err := New(
		WithLogger(logger),
		WithMode(ModeMigCaps),
		WithDriverRoot(hostRoot),
		WithEnabledHooks("chmod"),
	)
	require.NoError(t, err)

	deviceSpecs, err := lib.GetDeviceSpecsByID("config", "monitor")
	require.NoError(t, err)
	require.Len(t, deviceSpecs, 2)

	byName := make(map[string]specs.Device)
	for _, d := range deviceSpecs {
		byName[d.Name] = d
	}

	expected := map[string]string{
		"config":  "/dev/nvidia-caps/nvidia-cap1",
		"monitor": "/dev/nvidia-caps/nvidia-cap2",
	}
	for name, path := range expected {
		spec, ok := byName[name]
		require.Truef(t, ok, "expected a device named %q", name)

		require.Len(t, spec.ContainerEdits.DeviceNodes, 1)
		deviceNode := spec.ContainerEdits.DeviceNodes[0]
		require.Equal(t, path, deviceNode.Path)
		require.Equal(t, filepath.Join(hostRoot, path), deviceNode.HostPath)

		require.True(t, hasFolderPermissionHook(spec, "/dev/nvidia-caps"),
			"expected a folder-permission hook for /dev/nvidia-caps on device %q", name)
	}
}

func hasFolderPermissionHook(spec specs.Device, folder string) bool {
	for _, hook := range spec.ContainerEdits.Hooks {
		if slices.Contains(hook.Args, folder) {
			return true
		}
	}
	return false
}
