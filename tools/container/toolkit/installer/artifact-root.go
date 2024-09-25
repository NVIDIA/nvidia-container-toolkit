/**
# Copyright 2024 NVIDIA CORPORATION
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

package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

type artifactRoot struct {
	path        string
	libraries   lookup.Locator
	executables lookup.Locator
}

func newArtifactRoot(packageType string) (*artifactRoot, error) {
	path := "/"
	switch packageType {
	case "deb":
		path = "/artifacts/deb"
	case "rpm":
		path = "/artifacts/rpm"
	default:
		return nil, fmt.Errorf("invalid package type: %v", packageType)
	}

	a := artifactRoot{
		path: path,
		libraries: lookup.NewLibraryLocator(
			lookup.WithRoot(path),
			lookup.WithCount(1),
			lookup.WithSearchPaths(
				"/usr/lib64",
				"/usr/lib/x86_64-linux-gnu",
				"/usr/lib/aarch64-linux-gnu",
			),
		),
		executables: lookup.NewExecutableLocator(
			logger.New(),
			path,
		),
	}

	return &a, nil
}

func resolvePackageType(hostRoot string, packageType string) (rPackageTypes string, rerr error) {
	if packageType != "" && packageType != "auto" {
		return packageType, nil
	}

	if info, err := os.Stat(filepath.Join(hostRoot, "/usr/bin/rpm")); err != nil && !info.IsDir() {
		return "rpm", nil
	}
	if info, err := os.Stat(filepath.Join(hostRoot, "/usr/bin/dpkg")); err != nil && !info.IsDir() {
		return "deb", nil
	}

	return "deb", nil
}

func (r *artifactRoot) findLibrary(name string) (string, error) {
	candidates, err := r.libraries.Locate(name)
	if err != nil {
		return "", fmt.Errorf("error locating library: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("library %v not found", name)
	}

	return candidates[0], nil
}

func (r *artifactRoot) findExecutable(name string) (string, error) {
	candidates, err := r.executables.Locate(name)
	if err != nil {
		return "", fmt.Errorf("error locating executable: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("executable %v not found", name)
	}

	return candidates[0], nil
}
