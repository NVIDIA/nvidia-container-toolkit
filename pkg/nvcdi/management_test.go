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
	"os"
	"path/filepath"
	"slices"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

func TestManagementNVSDMLibraries(t *testing.T) {
	const version = "580.65.06"
	const libDir = "usr/lib/x86_64-linux-gnu"

	nvsdmLibs := []string{"libnvsdm.so.1", "libibumad.so.3", "libibmad.so.5"}
	allPaths := make([]string, len(nvsdmLibs))
	for i, lib := range nvsdmLibs {
		allPaths[i] = "/" + libDir + "/" + lib
	}

	testCases := map[string]struct {
		nvsdm         bool
		management    bool
		subRoot       string
		featureFlags  map[FeatureFlag]bool
		expectedPaths []string
	}{
		"nvsdm present in management spec": {
			nvsdm:         true,
			management:    true,
			expectedPaths: allPaths,
		},
		"nvsdm present is not mounted for non-management driver libraries": {
			nvsdm: true,
		},
		"nvsdm absent is not an error": {
			nvsdm:      false,
			management: true,
		},
		"nvsdm present in non-root driver root": {
			nvsdm:         true,
			management:    true,
			subRoot:       "driver-root",
			expectedPaths: allPaths,
		},
		"explicit driver libraries disabled": {
			nvsdm:         true,
			management:    true,
			featureFlags:  map[FeatureFlag]bool{FeatureEnableExplicitDriverLibraries: false},
			expectedPaths: allPaths,
		},
		"explicit driver libraries enabled": {
			nvsdm:         true,
			management:    true,
			featureFlags:  map[FeatureFlag]bool{FeatureEnableExplicitDriverLibraries: true},
			expectedPaths: allPaths,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			logger, _ := testlog.NewNullLogger()

			// Resolve symlinks (e.g. /var -> /private/var on macOS) since the
			// symlink locator returns resolved paths.
			tmpDir, err := filepath.EvalSymlinks(t.TempDir())
			require.NoError(t, err)
			driverRoot := filepath.Join(tmpDir, tc.subRoot)
			dir := filepath.Join(driverRoot, libDir)
			require.NoError(t, os.MkdirAll(dir, 0755))
			files := []string{"libcuda.so." + version, "libnvidia-ml.so." + version}
			if tc.nvsdm {
				files = append(files, nvsdmLibs...)
			}
			for _, f := range files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), nil, 0600))
			}

			l := &nvcdilib{
				logger:       logger,
				driver:       root.New(root.WithLogger(logger), root.WithDriverRoot(driverRoot)),
				featureFlags: tc.featureFlags,
				hookCreator:  discover.NewHookCreator(),
				editsFactory: edits.NewFactory(edits.WithLogger(logger)),
			}

			// Maps container path to host path for every mount in the edits.
			mounts := make(map[string]string)
			if tc.management {
				commonEdits, err := (*managementlib)(l).GetCommonEdits()
				require.NoError(t, err)
				for _, m := range commonEdits.Mounts {
					mounts[m.ContainerPath] = m.HostPath
				}
			} else {
				d, err := l.NewDriverLibraryDiscoverer(version)
				require.NoError(t, err)
				driverMounts, err := d.Mounts()
				require.NoError(t, err)
				for _, m := range driverMounts {
					mounts[m.Path] = m.HostPath
				}
			}

			var nvsdmPaths []string
			for containerPath, hostPath := range mounts {
				if slices.Contains(nvsdmLibs, filepath.Base(containerPath)) {
					nvsdmPaths = append(nvsdmPaths, containerPath)
					require.Equal(t, filepath.Join(driverRoot, containerPath), hostPath)
				}
			}
			require.ElementsMatch(t, tc.expectedPaths, nvsdmPaths)
		})
	}
}
