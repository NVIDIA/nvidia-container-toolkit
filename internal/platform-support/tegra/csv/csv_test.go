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

package csv

import (
	"path/filepath"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetFileList(t *testing.T) {
	moduleRoot, _ := test.GetModuleRoot()

	testCases := []struct {
		description   string
		root          string
		files         []string
		expectedError error
	}{
		{
			description: "returns list of CSV files",
			root:        "test/input/csv_samples/",
			files: []string{
				"jetson.csv",
				"simple_wrong.csv",
				"simple.csv",
				"spaced.csv",
			},
		},
		{
			description: "handles empty folder",
			root:        "test/input/csv_samples/empty",
		},
		{
			description: "handles non-existent folder",
			root:        "test/input/csv_samples/NONEXISTENT",
		},
		{
			description: "handles non-existent folder root",
			root:        "/NONEXISTENT/test/input/csv_samples/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			root := filepath.Join(moduleRoot, tc.root)
			files, err := GetFileList(root)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Empty(t, files)
				return
			}

			require.NoError(t, err)

			var foundFiles []string
			for _, f := range files {
				require.Equal(t, root, filepath.Dir(f))
				require.Equal(t, ".csv", filepath.Ext(f))
				foundFiles = append(foundFiles, filepath.Base(f))
			}

			require.ElementsMatch(t, tc.files, foundFiles)
		})
	}
}
