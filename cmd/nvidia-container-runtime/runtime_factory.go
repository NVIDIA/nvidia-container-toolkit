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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/runtime"
	"github.com/sirupsen/logrus"
)

// newNVIDIAContainerRuntime is a factory method that constructs a runtime based on the selected configuration and specified logger
func newNVIDIAContainerRuntime(logger *logrus.Logger, cfg *config.Config, argv []string) (oci.Runtime, error) {
	lowLevelRuntime, err := oci.NewLowLevelRuntime(logger, cfg.NVIDIAContainerRuntimeConfig.Runtimes)
	if err != nil {
		return nil, fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	if !oci.HasCreateSubcommand(argv) {
		logger.Debugf("Skipping modifier for non-create subcommand")
		return lowLevelRuntime, nil
	}

	ociSpec, err := oci.NewSpec(logger, argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	specModifier, err := newSpecModifier(logger, cfg, ociSpec, argv)
	if err != nil {
		return nil, fmt.Errorf("failed to construct OCI spec modifier: %v", err)
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

// newSpecModifier is a factory method that creates constructs an OCI spec modifer based on the provided config.
func newSpecModifier(logger *logrus.Logger, cfg *config.Config, ociSpec oci.Spec, argv []string) (oci.SpecModifier, error) {
	modeModifier, err := newModeModifier(logger, cfg, ociSpec, argv)
	if err != nil {
		return nil, err
	}

	gdsModifier, err := modifier.NewGDSModifier(logger, cfg, ociSpec)
	if err != nil {
		return nil, err
	}

	mofedModifier, err := modifier.NewMOFEDModifier(logger, cfg, ociSpec)
	if err != nil {
		return nil, err
	}

	modifiers := modifier.Merge(
		modeModifier,
		gdsModifier,
		mofedModifier,
	)
	return modifiers, nil
}

func newModeModifier(logger *logrus.Logger, cfg *config.Config, ociSpec oci.Spec, argv []string) (oci.SpecModifier, error) {
	switch info.ResolveAutoMode(logger, cfg.NVIDIAContainerRuntimeConfig.Mode) {
	case "legacy":
		return modifier.NewStableRuntimeModifier(logger), nil
	case "csv":
		return modifier.NewCSVModifier(logger, cfg, ociSpec)
	}

	return nil, fmt.Errorf("invalid runtime mode: %v", cfg.NVIDIAContainerRuntimeConfig.Mode)
}
