/*
*
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
*
*/
package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

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

func defaultSourceRoot(hostRoot string, packageType string) (string, error) {
	resolvedPackageType, err := resolvePackageType(hostRoot, packageType)
	if err != nil {
		return "", err
	}
	switch resolvedPackageType {
	case "deb":
		return "/artifacts/deb", nil
	case "rpm":
		return "/artifacts/rpm", nil
	default:
		return "", fmt.Errorf("invalid package type: %v", resolvedPackageType)
	}
}
