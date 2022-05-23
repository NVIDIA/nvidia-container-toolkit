/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package info

import (
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

// HasNVML returns true if NVML is detected on the sytems
func HasNVML() (bool, string) {
	const (
		nvmlLibraryName      = "libnvidia-ml.so.1"
		nvmlLibraryLoadFlags = dl.RTLD_LAZY
	)
	lib := dl.New(nvmlLibraryName, nvmlLibraryLoadFlags)
	if err := lib.Open(); err != nil {
		return false, fmt.Sprintf("could not load NVML: %v", err)
	}
	defer lib.Close()

	return true, "found NVML library"
}

// IsTegraSystem returns true if the system is detected as a Tegra-based system
func IsTegraSystem() (bool, string) {
	const tegraReleaseFile = "/etc/nv_tegra_release"
	const tegraFamilyFile = "/sys/devices/soc0/family"

	if info, err := os.Stat(tegraReleaseFile); err == nil && !info.IsDir() {
		return true, fmt.Sprintf("%v found", tegraReleaseFile)
	}

	if info, err := os.Stat(tegraFamilyFile); err != nil || info.IsDir() {
		return false, fmt.Sprintf("%v file not found", tegraFamilyFile)
	}

	contents, err := os.ReadFile(tegraFamilyFile)
	if err != nil {
		return false, fmt.Sprintf("could not read %v", tegraFamilyFile)
	}

	if strings.HasPrefix(strings.ToLower(string(contents)), "tegra") {
		return true, fmt.Sprintf("%v has 'tegra' prefix", tegraFamilyFile)
	}

	return false, fmt.Sprintf("%v has no 'tegra' prefix", tegraFamilyFile)
}
