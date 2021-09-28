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
	ociSpecFileName          = "config.json"
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
)

// newRuntime is a factory method that constructs a runtime based on the selected configuration.
func newRuntime(argv []string) (oci.Runtime, error) {
	ociSpec, err := newOCISpec(argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	runc, err := newRuncRuntime()
	if err != nil {
		return nil, fmt.Errorf("error constructing runc runtime: %v", err)
	}

	r, err := newNvidiaContainerRuntimeWithLogger(logger.Logger, runc, ociSpec)
	if err != nil {
		return nil, fmt.Errorf("error constructing NVIDIA Container Runtime: %v", err)
	}

	return r, nil
}

// newOCISpec constructs an OCI spec for the provided arguments
func newOCISpec(argv []string) (oci.Spec, error) {
	bundleDir, err := oci.GetBundleDir(argv)
	if err != nil {
		return nil, fmt.Errorf("error parsing command line arguments: %v", err)
	}
	logger.Infof("Using bundle directory: %v", bundleDir)

	ociSpecPath := oci.GetSpecFilePath(bundleDir)
	logger.Infof("Using OCI specification file path: %v", ociSpecPath)

	ociSpec := oci.NewSpecFromFile(ociSpecPath)

	return ociSpec, nil
}

// newRuncRuntime locates the runc binary and wraps it in a SyscallExecRuntime
func newRuncRuntime() (oci.Runtime, error) {
	return oci.NewLowLevelRuntimeWithLogger(
		logger.Logger,
		dockerRuncExecutableName,
		runcExecutableName,
	)
}
