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

package main

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
)

// newRuntime is a factory method that constructs a runtime based on the selected configuration.
func newRuntime(argv []string) (oci.Runtime, error) {
	ociSpec, err := oci.NewSpec(logger.Logger, argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	lowLevelRuntimeCandidates := []string{dockerRuncExecutableName, runcExecutableName}
	lowLevelRuntime, err := oci.NewLowLevelRuntime(logger.Logger, lowLevelRuntimeCandidates)
	if err != nil {
		return nil, fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	r, err := newNvidiaContainerRuntime(logger.Logger, lowLevelRuntime, ociSpec)
	if err != nil {
		return nil, fmt.Errorf("error constructing NVIDIA Container Runtime: %v", err)
	}

	return r, nil
}
