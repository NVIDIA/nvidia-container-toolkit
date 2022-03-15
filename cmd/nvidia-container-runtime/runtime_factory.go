/*
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-container-runtime/modifier"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/runtime"
	"github.com/sirupsen/logrus"
)

const (
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
)

// newNVIDIAContainerRuntime is a factory method that constructs a runtime based on the selected configuration and specified logger
func newNVIDIAContainerRuntime(logger *logrus.Logger, cfg *config.RuntimeConfig, argv []string) (oci.Runtime, error) {
	ociSpec, err := oci.NewSpec(logger, argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	lowLevelRuntimeCandidates := []string{dockerRuncExecutableName, runcExecutableName}
	lowLevelRuntime, err := oci.NewLowLevelRuntime(logger, lowLevelRuntimeCandidates)
	if err != nil {
		return nil, fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	var specModifier oci.SpecModifier
	if cfg.Experimental {
		specModifier, err = modifier.NewExperimentalModifier(logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to construct experimental modifier: %v", err)
		}
	} else {
		specModifier = modifier.NewStableRuntimeModifier(logger)
	}

	// Create the wrapping runtime with the specified modifier
	r := runtime.NewModifyingRuntimeWrapper(
		logger,
		lowLevelRuntime,
		ociSpec,
		specModifier,
	)

	return r, nil
}
