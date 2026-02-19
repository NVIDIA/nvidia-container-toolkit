/**
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
**/

package tegra

import (
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

func TestDiscovererFromCSVFiles(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description         string
		moutSpecs           MountSpecPathsByType
		ignorePatterns      []string
		symlinkLocator      lookup.Locator
		symlinkChainLocator lookup.Locator
		symlinkResolver     func(string) (string, error)
		expectedError       error
		expectedMounts      []discover.Mount
		expectedMountsError error
		expectedHooks       []discover.Hook
		expectedHooksError  error
	}{
		{
			// TODO: This current resolves to two mounts that are the same.
			// These are deduplicated at a later stage. We could consider deduplicating earlier in the pipeline.
			description: "symlink is resolved to target; mounts and symlink are created",
			moutSpecs: MountSpecPathsByType{
				"lib": {"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"},
				"sym": {"/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so"},
			},
			symlinkLocator: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
						return []string{"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"}, nil
					}
					return []string{path}, nil
				},
			},
			symlinkChainLocator: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
						return []string{"/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so", "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"}, nil
					}
					return nil, fmt.Errorf("Unexpected path: %v", path)
				},
			},
			symlinkResolver: func(path string) (string, error) {
				if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
					return "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so", nil
				}
				return path, nil
			},
			expectedMounts: []discover.Mount{
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{
						"nvidia-cdi-hook",
						"create-symlinks",
						"--link",
						"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so::/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so",
					},
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
		{
			// TODO: This current resolves to two mounts that are the same.
			// These are deduplicated at a later stage. We could consider deduplicating earlier in the pipeline.
			description: "single glob filter does not remove symlink mounts",
			moutSpecs: MountSpecPathsByType{
				"lib": {"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"},
				"sym": {"/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so"},
			},
			ignorePatterns: []string{"*.so"},
			symlinkLocator: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
						return []string{"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"}, nil
					}
					return []string{path}, nil
				},
			},
			symlinkChainLocator: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
						return []string{"/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so", "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"}, nil
					}
					return nil, fmt.Errorf("Unexpected path: %v", path)
				},
			},
			symlinkResolver: func(path string) (string, error) {
				if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
					return "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so", nil
				}
				return path, nil
			},
			expectedMounts: []discover.Mount{
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-cdi-hook",
					Args: []string{
						"nvidia-cdi-hook",
						"create-symlinks",
						"--link",
						"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so::/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so",
					},
					Env: []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
		{
			description: "** filter removes symlink mounts",
			moutSpecs: MountSpecPathsByType{
				"lib": {"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"},
				"sym": {"/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so"},
			},
			symlinkLocator: &lookup.LocatorMock{
				LocateFunc: func(path string) ([]string, error) {
					if path == "/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so" {
						return []string{"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so"}, nil
					}
					return []string{path}, nil
				},
			},
			ignorePatterns: []string{"**/*.so"},
			expectedMounts: []discover.Mount{
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
			},
		},
	}

	hookCreator := discover.NewHookCreator()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			o := options{
				logger:              logger,
				hookCreator:         hookCreator,
				symlinkLocator:      tc.symlinkLocator,
				symlinkChainLocator: tc.symlinkChainLocator,
				resolveSymlink:      tc.symlinkResolver,

				mountSpecs: Transform(
					tc.moutSpecs,
					IgnoreSymlinkMountSpecsByPattern(tc.ignorePatterns...),
				),
			}

			d := o.newDiscovererFromMountSpecs(o.mountSpecs.MountSpecPathsByType())

			hooks, err := d.Hooks()
			require.ErrorIs(t, err, tc.expectedHooksError)
			require.EqualValues(t, tc.expectedHooks, hooks)

			mounts, err := d.Mounts()
			require.ErrorIs(t, err, tc.expectedMountsError)
			require.EqualValues(t, tc.expectedMounts, mounts)

		})
	}
}
