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

// Logger is a basic interface for logging to allow these functions to be called
// from code where logrus is not used.
type Logger interface {
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
}

// ResolveAutoMode determines the correct mode for the platform if set to "auto"
func ResolveAutoMode(logger Logger, mode string) (rmode string) {
	if mode != "auto" {
		return mode
	}
	defer func() {
		logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	isTegra, reason := IsTegraSystem()
	logger.Debugf("Is Tegra-based system? %v: %v", isTegra, reason)

	hasNVML, reason := HasNVML()
	logger.Debugf("Has NVML? %v: %v", hasNVML, reason)

	if isTegra && !hasNVML {
		return "csv"
	}

	return "legacy"
}
