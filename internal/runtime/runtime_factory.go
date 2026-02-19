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

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// newNVIDIAContainerRuntime is a factory method that constructs a runtime based on the selected configuration and specified logger
func newNVIDIAContainerRuntime(logger logger.Interface, driver *root.Driver, cfg *config.Config, argv []string) (oci.Runtime, error) {
	lowLevelRuntime, err := oci.NewLowLevelRuntime(logger, cfg.NVIDIAContainerRuntimeConfig.Runtimes)
	if err != nil {
		return nil, fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	logger.Tracef("Using low-level runtime %v", lowLevelRuntime.String())
	if !oci.HasCreateSubcommand(argv) {
		logger.Tracef("Skipping modifier for non-create subcommand")
		return lowLevelRuntime, nil
	}

	ociSpec, err := oci.NewSpec(argv,
		oci.WithLogger(logger),
		oci.WithAllowUnknownFields(cfg.Features.AllowUnknownOCISpecFields.IsEnabled()),
	)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	specModifier, err := newSpecModifier(logger, driver, cfg, ociSpec)
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
func newSpecModifier(logger logger.Interface, driver *root.Driver, cfg *config.Config, ociSpec oci.Spec) (oci.SpecModifier, error) {
	mode, image, err := initRuntimeModeAndImage(logger, cfg, ociSpec)
	if err != nil {
		return nil, err
	}

	hookCreator := discover.NewHookCreator(discover.WithNVIDIACDIHookPath(cfg.NVIDIACTKConfig.Path))
	return modifier.New(
		modifier.WithLogger(logger),
		modifier.WithConfig(cfg),
		modifier.WithImage(image),
		modifier.WithDriver(driver),
		modifier.WithHookCreator(hookCreator),
		modifier.WithRuntimeMode(mode),
	)
}

// initRuntimeModeAndImage constructs an image from the specified OCI runtime
// specification and runtime config.
// The image is also used to determine the runtime mode to apply.
// If a non-CDI mode is detected we ensure that the image does not process
// annotation devices.
func initRuntimeModeAndImage(logger logger.Interface, cfg *config.Config, ociSpec oci.Spec) (info.RuntimeMode, *image.CUDA, error) {
	rawSpec, err := ociSpec.Load()
	if err != nil {
		return "", nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	image, err := image.NewCUDAImageFromSpec(
		rawSpec,
		image.WithLogger(logger),
		image.WithAcceptDeviceListAsVolumeMounts(cfg.AcceptDeviceListAsVolumeMounts),
		image.WithAcceptEnvvarUnprivileged(cfg.AcceptEnvvarUnprivileged),
		image.WithAnnotationsPrefixes(cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.AnnotationPrefixes...),
		image.WithPreferredVisibleDevicesEnvVars(cfg.SwarmResource),
		image.WithIgnoreImexChannelRequests(cfg.Features.IgnoreImexChannelRequests.IsEnabled()),
	)
	if err != nil {
		return "", nil, err
	}

	modeResolver := info.NewRuntimeModeResolver(
		info.WithLogger(logger),
		info.WithImage(&image),
	)
	mode := modeResolver.ResolveRuntimeMode(cfg.NVIDIAContainerRuntimeConfig.Mode)
	// We update the mode here so that we can continue passing just the config to other functions.
	cfg.NVIDIAContainerRuntimeConfig.Mode = string(mode)

	if mode == "cdi" || len(cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.AnnotationPrefixes) == 0 {
		return mode, &image, nil
	}

	// For non-cdi modes we explicitly set the annotation prefixes to nil and
	// call this function again to force a reconstruction of the image.
	// Note that since the mode is now explicitly set, we will effectively skip
	// the mode resolution.
	cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.AnnotationPrefixes = nil

	return initRuntimeModeAndImage(logger, cfg, ociSpec)
}
