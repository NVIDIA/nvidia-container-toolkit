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
*/

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapper(t *testing.T) {
	const shebang = "#! /bin/sh"
	const destFolder = "/dest/folder"
	const dotfileName = "source.real"

	testCases := []struct {
		e             executable
		expectedLines []string
	}{
		{
			e: executable{},
			expectedLines: []string{
				shebang,
				"PATH=/dest/folder:$PATH \\",
				"source.real \\",
				"\t\"$@\"",
				"",
			},
		},
		{
			e: executable{
				env: map[string]string{
					"PATH": "some-path",
				},
			},
			expectedLines: []string{
				shebang,
				"PATH=/dest/folder:some-path \\",
				"source.real \\",
				"\t\"$@\"",
				"",
			},
		},
		{
			e: executable{
				preLines: []string{
					"preline1",
					"preline2",
				},
			},
			expectedLines: []string{
				shebang,
				"preline1",
				"preline2",
				"PATH=/dest/folder:$PATH \\",
				"source.real \\",
				"\t\"$@\"",
				"",
			},
		},
		{
			e: executable{
				argLines: []string{
					"argline1",
					"argline2",
				},
			},
			expectedLines: []string{
				shebang,
				"PATH=/dest/folder:$PATH \\",
				"source.real \\",
				"\targline1 \\",
				"\targline2 \\",
				"\t\"$@\"",
				"",
			},
		},
	}

	for i, tc := range testCases {
		buf := &bytes.Buffer{}

		err := tc.e.writeWrapperTo(buf, destFolder, dotfileName)
		require.NoError(t, err)

		exepectedContents := strings.Join(tc.expectedLines, "\n")
		require.Equal(t, exepectedContents, buf.String(), "%v: %v", i, tc)
	}
}

func TestInstallExecutable(t *testing.T) {
	inputFolder, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(inputFolder)

	// Create the source file
	source := filepath.Join(inputFolder, "input")
	sourceFile, err := os.Create(source)

	base := filepath.Base(source)

	require.NoError(t, err)
	require.NoError(t, sourceFile.Close())

	e := executable{
		source: source,
		target: executableTarget{
			dotfileName: "input.real",
			wrapperName: "input",
		},
	}

	destFolder, err := os.MkdirTemp("", "output-*")
	require.NoError(t, err)
	defer os.RemoveAll(destFolder)

	installed, err := e.install(destFolder)

	require.NoError(t, err)
	require.Equal(t, filepath.Join(destFolder, base), installed)

	// Now check the post conditions:
	sourceInfo, err := os.Stat(source)
	require.NoError(t, err)

	destInfo, err := os.Stat(filepath.Join(destFolder, base+".real"))
	require.NoError(t, err)
	require.Equal(t, sourceInfo.Size(), destInfo.Size())
	require.Equal(t, sourceInfo.Mode(), destInfo.Mode())

	wrapperInfo, err := os.Stat(installed)
	require.NoError(t, err)
	require.NotEqual(t, 0, wrapperInfo.Mode()&0111)
}
