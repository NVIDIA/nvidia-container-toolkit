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

package ocihook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateHook creates an OCI hook file for the specified NVIDIA Container Runtime hook path
func CreateHook(hookFilePath string, nvidiaContainerRuntimeHookExecutablePath string) error {
	var output io.Writer
	if hookFilePath == "" {
		output = os.Stdout
	} else {
		if hooksDir := filepath.Dir(hookFilePath); hooksDir != "" {
			err := os.MkdirAll(hooksDir, 0755)
			if err != nil {
				return fmt.Errorf("error creating hooks directory %v: %v", hooksDir, err)
			}
		}

		hookFile, err := os.Create(hookFilePath)
		if err != nil {
			return fmt.Errorf("error creating hook file '%v': %v", hookFilePath, err)
		}
		defer hookFile.Close()
		output = hookFile
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(generateOciHook(nvidiaContainerRuntimeHookExecutablePath)); err != nil {
		return fmt.Errorf("error writing hook file: %v", err)
	}
	return nil
}

func generateOciHook(executablePath string) podmanHook {
	pathParts := []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}

	dir := filepath.Dir(executablePath)
	var found bool
	for _, pathPart := range pathParts {
		if pathPart == dir {
			found = true
			break
		}
	}
	if !found {
		pathParts = append(pathParts, dir)
	}

	envPath := "PATH=" + strings.Join(pathParts, ":")
	always := true

	hook := podmanHook{
		Version: "1.0.0",
		Stages:  []string{"prestart"},
		Hook: specHook{
			Path: executablePath,
			Args: []string{filepath.Base(executablePath), "prestart"},
			Env:  []string{envPath},
		},
		When: When{
			Always:   &always,
			Commands: []string{".*"},
		},
	}
	return hook
}
