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

package runtime

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// newNVIDIAContainerRuntime is a factory method that constructs a runtime based on the selected configuration and specified logger
func newNVIDIAContainerRuntime(logger logger.Interface, cfg *config.Config, argv []string, driver *root.Driver) (oci.Runtime, error) {
	lowLevelRuntime, err := oci.NewLowLevelRuntime(logger, cfg.NVIDIAContainerRuntimeConfig.Runtimes)
	if err != nil {
		return nil, fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	logger.Tracef("Using low-level runtime %v", lowLevelRuntime.String())
	if !oci.HasCreateSubcommand(argv) {
		logger.Tracef("Skipping modifier for non-create subcommand")
		return lowLevelRuntime, nil
	}

	ociSpec, err := oci.NewSpec(logger, argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	specModifier, err := newSpecModifier(logger, cfg, ociSpec, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to construct OCI spec modifier: %v", err)
	}

	// Create the wrapping runtime with the specified modifier.
	r := oci.NewModifyingRuntimeWrapper(
		logger,
		lowLevelRuntime,
		ociSpec,
		specModifier,
	)

	return r, nil
}

// newSpecModifier is a factory method that creates constructs an OCI spec modifer based on the provided config.
func newSpecModifier(logger logger.Interface, cfg *config.Config, ociSpec oci.Spec, driver *root.Driver) (oci.SpecModifier, error) {
	rawSpec, err := ociSpec.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	image, err := image.NewCUDAImageFromSpec(rawSpec)
	if err != nil {
		return nil, err
	}

	mode := info.ResolveAutoMode(logger, cfg.NVIDIAContainerRuntimeConfig.Mode, image)
	// We update the mode here so that we can continue passing just the config to other functions.
	cfg.NVIDIAContainerRuntimeConfig.Mode = mode
	modeModifier, err := newModeModifier(logger, mode, cfg, ociSpec, image)
	if err != nil {
		return nil, err
	}

	var modifiers modifier.List
	for _, modifierType := range supportedModifierTypes(mode) {
		switch modifierType {
		case "mode":
			modifiers = append(modifiers, modeModifier)
		case "nvidia-hook-remover":
			modifiers = append(modifiers, modifier.NewNvidiaContainerRuntimeHookRemover(logger))
		case "graphics":
			graphicsModifier, err := modifier.NewGraphicsModifier(logger, cfg, image, driver)
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, graphicsModifier)
		case "feature-gated":
			featureGatedModifier, err := modifier.NewFeatureGatedModifier(logger, cfg, image, driver)
			if err != nil {
				return nil, err
			}
			modifiers = append(modifiers, featureGatedModifier)
		}
	}

	return modifiers, nil
}

func newModeModifier(logger logger.Interface, mode string, cfg *config.Config, ociSpec oci.Spec, image image.CUDA) (oci.SpecModifier, error) {
	switch mode {
	case "legacy":
		return modifier.NewStableRuntimeModifier(logger, cfg.NVIDIAContainerRuntimeHookConfig.Path), nil
	case "csv":
		return modifier.NewCSVModifier(logger, cfg, image)
	case "cdi":
		return modifier.NewCDIModifier(logger, cfg, ociSpec)
	}

	return nil, fmt.Errorf("invalid runtime mode: %v", cfg.NVIDIAContainerRuntimeConfig.Mode)
}

// supportedModifierTypes returns the modifiers supported for a specific runtime mode.
func supportedModifierTypes(mode string) []string {
	switch mode {
	case "cdi":
		// For CDI mode we make no additional modifications.
		return []string{"nvidia-hook-remover", "mode"}
	case "csv":
		// For CSV mode we support mode and feature-gated modification.
		return []string{"nvidia-hook-remover", "feature-gated", "mode"}
	default:
		return []string{"feature-gated", "graphics", "mode"}
	}
}
