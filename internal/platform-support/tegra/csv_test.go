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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

func TestDiscovererFromCSVFiles(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description         string
		moutSpecs           map[csv.MountSpecType][]string
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
			moutSpecs: map[csv.MountSpecType][]string{
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
					Options:  []string{"ro", "nosuid", "nodev", "bind"},
				},
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "bind"},
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-ctk",
					Args: []string{
						"nvidia-ctk",
						"hook",
						"create-symlinks",
						"--link",
						"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so::/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so",
					},
				},
			},
		},
		{
			// TODO: This current resolves to two mounts that are the same.
			// These are deduplicated at a later stage. We could consider deduplicating earlier in the pipeline.
			description: "single glob filter does not remove symlink mounts",
			moutSpecs: map[csv.MountSpecType][]string{
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
					Options:  []string{"ro", "nosuid", "nodev", "bind"},
				},
				{
					Path:     "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					HostPath: "/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so",
					Options:  []string{"ro", "nosuid", "nodev", "bind"},
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "/usr/bin/nvidia-ctk",
					Args: []string{
						"nvidia-ctk",
						"hook",
						"create-symlinks",
						"--link",
						"/usr/lib/aarch64-linux-gnu/tegra/libv4l2_nvargus.so::/usr/lib/aarch64-linux-gnu/libv4l/plugins/nv/libv4l2_nvargus.so",
					},
				},
			},
		},
		{
			description: "** filter removes symlink mounts",
			moutSpecs: map[csv.MountSpecType][]string{
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
					Options:  []string{"ro", "nosuid", "nodev", "bind"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			defer setGetTargetsFromCSVFiles(tc.moutSpecs)()

			o := tegraOptions{
				logger:              logger,
				nvidiaCTKPath:       "/usr/bin/nvidia-ctk",
				csvFiles:            []string{"dummy"},
				ignorePatterns:      tc.ignorePatterns,
				symlinkLocator:      tc.symlinkLocator,
				symlinkChainLocator: tc.symlinkChainLocator,
				resolveSymlink:      tc.symlinkResolver,
			}

			d, err := o.newDiscovererFromCSVFiles()
			require.ErrorIs(t, err, tc.expectedError)

			hooks, err := d.Hooks()
			require.ErrorIs(t, err, tc.expectedHooksError)
			require.EqualValues(t, tc.expectedHooks, hooks)

			mounts, err := d.Mounts()
			require.ErrorIs(t, err, tc.expectedMountsError)
			require.EqualValues(t, tc.expectedMounts, mounts)

		})
	}
}

func setGetTargetsFromCSVFiles(ovverride map[csv.MountSpecType][]string) func() {
	original := getTargetsFromCSVFiles
	getTargetsFromCSVFiles = func(logger logger.Interface, files []string) map[csv.MountSpecType][]string {
		return ovverride
	}

	return func() {
		getTargetsFromCSVFiles = original
	}
}
