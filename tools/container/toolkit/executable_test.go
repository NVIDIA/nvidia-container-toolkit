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
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapper(t *testing.T) {
	createTestWrapperProgram(t)

	testCases := []struct {
		e            executable
		expectedArgv []string
		expectedEnvv []string
	}{
		{
			e: executable{source: "source"},
			expectedEnvv: []string{
				fmt.Sprintf("<PATH=%s", destDirPattern),
			},
		},
		{
			e: executable{
				source: "source",
				envm: map[string]string{
					"FOO": "BAR",
				},
			},
			expectedEnvv: []string{
				fmt.Sprintf("<PATH=%s", destDirPattern),
				"FOO=BAR",
			},
		},
		{
			e: executable{
				source: "source",
				envm: map[string]string{
					"PATH": "some-path",
					"FOO":  "BAR",
				},
			},
			expectedEnvv: []string{
				"FOO=BAR",
				fmt.Sprintf("PATH=%s:some-path", destDirPattern),
			},
		},
		{
			e: executable{
				source: "source",
				argv: []string{
					"argb",
					"arga",
					"argc",
				},
			},
			expectedArgv: []string{
				"argb",
				"arga",
				"argc",
			},
			expectedEnvv: []string{
				fmt.Sprintf("<PATH=%s", destDirPattern),
			},
		},
	}

	for _, tc := range testCases {
		destFolder := t.TempDir()
		r := newReplacements(destDirPattern, destFolder)
		for k, v := range tc.expectedEnvv {
			tc.expectedEnvv[k] = r.apply(v)
		}
		path, err := tc.e.installWrapper(destFolder)
		require.NoError(t, err)
		require.FileExists(t, path)
		envv, err := readAllLines(path + ".envv")
		require.NoError(t, err)
		require.Equal(t, tc.expectedEnvv, envv)
		argv, err := readAllLines(path + ".argv")
		if tc.expectedArgv == nil {
			require.ErrorAs(t, err, &fs.ErrNotExist)
		} else {
			require.Equal(t, tc.expectedArgv, argv)

		}
	}
}

func TestInstallExecutable(t *testing.T) {
	createTestWrapperProgram(t)

	// Create the source file
	source := filepath.Join(t.TempDir(), "input")
	sourceFile, err := os.Create(source)

	base := filepath.Base(source)

	require.NoError(t, err)
	require.NoError(t, sourceFile.Close())

	e := executable{
		source: source,
		target: executableTarget{
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

func createTestWrapperProgram(t *testing.T) {
	t.Helper()
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("error getting current executable: %v", err)
	}
	wrapperPath := filepath.Join(filepath.Dir(currentExe), "wrapper")
	f, err := os.Create(wrapperPath)
	if err != nil {
		t.Fatalf("error creating test wrapper: %v", err)
	}
	f.Close()
}

func readAllLines(path string) (s []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s = append(s, scanner.Text())
	}
	err = scanner.Err()
	return
}
