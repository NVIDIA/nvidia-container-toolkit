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

package engine

import "strings"

// GetBinaryPathsForRuntimes returns the list of binary paths for common runtimes.
// The following list of runtimes is considered:
//
//	the default runtime, "runc", and "crun"
//
// If an nvidia* runtime is set as the default runtime, this is ignored.
func GetBinaryPathsForRuntimes(cfg Interface) []string {

	var binaryPaths []string
	seen := make(map[string]bool)
	for _, runtime := range GetLowLevelRuntimes(cfg) {
		runtimeConfig, err := cfg.GetRuntimeConfig(runtime)
		if err != nil {
			// TODO: It will be useful to log the error when GetRuntimeConfig fails for a runtime
			continue
		}
		binaryPath := runtimeConfig.GetBinaryPath()
		if binaryPath == "" || seen[binaryPath] {
			continue
		}
		seen[binaryPath] = true
		binaryPaths = append(binaryPaths, binaryPath)
	}

	return binaryPaths
}

// GetLowLevelRuntimes returns a predefined list low-level runtimes from the specified config.
// nvidia* runtimes are ignored.
func GetLowLevelRuntimes(cfg Interface) []string {
	var runtimes []string
	isValidDefault := func(s string) bool {
		if s == "" {
			return false
		}
		// ignore nvidia* runtimes.
		return !strings.HasPrefix(s, "nvidia")
	}
	if defaultRuntime := cfg.DefaultRuntime(); isValidDefault(defaultRuntime) {
		runtimes = append(runtimes, defaultRuntime)
	}
	return append(runtimes, "runc", "crun")
}
