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
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

func TestNVSDMLibraryDiscovery(t *testing.T) {
	const version = "580.65.06"
	const libDir = "usr/lib/x86_64-linux-gnu"

	testCases := map[string]struct {
		nvsdm         bool
		subRoot       string
		featureFlags  map[FeatureFlag]bool
		expectedPaths []string
	}{
		"nvsdm present": {
			nvsdm:         true,
			expectedPaths: []string{"/" + libDir + "/libnvsdm.so.1"},
		},
		"nvsdm absent is not an error": {
			nvsdm: false,
		},
		"nvsdm present in non-root driver root": {
			nvsdm:         true,
			subRoot:       "driver-root",
			expectedPaths: []string{"/" + libDir + "/libnvsdm.so.1"},
		},
		"explicit driver libraries disabled": {
			nvsdm:         true,
			featureFlags:  map[FeatureFlag]bool{FeatureEnableExplicitDriverLibraries: false},
			expectedPaths: []string{"/" + libDir + "/libnvsdm.so.1"},
		},
		"explicit driver libraries enabled": {
			nvsdm:         true,
			featureFlags:  map[FeatureFlag]bool{FeatureEnableExplicitDriverLibraries: true},
			expectedPaths: []string{"/" + libDir + "/libnvsdm.so.1"},
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
				files = append(files, "libnvsdm.so.1")
			}
			for _, f := range files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), nil, 0600))
			}

			l := &nvcdilib{
				logger:       logger,
				driver:       root.New(root.WithLogger(logger), root.WithDriverRoot(driverRoot)),
				featureFlags: tc.featureFlags,
				hookCreator:  discover.NewHookCreator(),
			}

			d, err := l.NewDriverLibraryDiscoverer(version)
			require.NoError(t, err)
			mounts, err := d.Mounts()
			require.NoError(t, err)

			var nvsdmPaths []string
			for _, m := range mounts {
				if filepath.Base(m.Path) == "libnvsdm.so.1" {
					nvsdmPaths = append(nvsdmPaths, m.Path)
					require.Equal(t, filepath.Join(driverRoot, m.Path), m.HostPath)
				}
			}
			require.Equal(t, tc.expectedPaths, nvsdmPaths)
		})
	}
}
