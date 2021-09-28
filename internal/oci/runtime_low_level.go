/*
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

package oci

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// NewLowLevelRuntime creates a Runtime that wraps a low-level runtime executable.
// The executable specified is taken from the list of supplied candidates, with the first match
// present in the PATH being selected.
func NewLowLevelRuntime(candidates ...string) (Runtime, error) {
	return NewLowLevelRuntimeWithLogger(log.StandardLogger(), candidates...)
}

// NewLowLevelRuntimeWithLogger creates a Runtime as with NewLowLevelRuntime using the specified logger.
func NewLowLevelRuntimeWithLogger(logger *log.Logger, candidates ...string) (Runtime, error) {
	runtimePath, err := findRuntime(candidates)
	if err != nil {
		return nil, fmt.Errorf("error locating runtime: %v", err)
	}

	return NewRuntimeForPathWithLogger(logger, runtimePath)
}

// findRuntime checks elements in a list of supplied candidates for a matching executable in the PATH.
// The absolute path to the first match is returned.
func findRuntime(candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("at least one runtime candidate must be specified")
	}

	for _, candidate := range candidates {
		log.Infof("Looking for runtime binary '%v'", candidate)
		runcPath, err := exec.LookPath(candidate)
		if err == nil {
			log.Infof("Found runtime binary '%v'", runcPath)
			return runcPath, nil
		}
		log.Warnf("Runtime binary '%v' not found: %v", candidate, err)
	}

	return "", fmt.Errorf("no runtime binary found from candidate list: %v", candidates)
}
