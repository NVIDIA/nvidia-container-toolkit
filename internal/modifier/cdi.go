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

package modifier

import (
	"fmt"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier/cdi"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
)

// NewCDIModifier creates an OCI spec modifier that determines the modifications to make based on the
// CDI specifications available on the system. The NVIDIA_VISIBLE_DEVICES environment variable is
// used to select the devices to include.
func NewCDIModifier(logger logger.Interface, cfg *config.Config, ociSpec oci.Spec) (oci.SpecModifier, error) {
	devices, err := getDevicesFromSpec(logger, ociSpec, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get required devices from OCI specification: %v", err)
	}
	if len(devices) == 0 {
		logger.Debugf("No devices requested; no modification required.")
		return nil, nil
	}
	logger.Debugf("Creating CDI modifier for devices: %v", devices)

	automaticDevices := filterAutomaticDevices(devices)
	if len(automaticDevices) != len(devices) && len(automaticDevices) > 0 {
		return nil, fmt.Errorf("requesting a CDI device with vendor 'runtime.nvidia.com' is not supported when requesting other CDI devices")
	}
	if len(automaticDevices) > 0 {
		automaticModifier, err := newAutomaticCDISpecModifier(logger, cfg, automaticDevices)
		if err == nil {
			return automaticModifier, nil
		}
		logger.Warningf("Failed to create the automatic CDI modifier: %w", err)
		logger.Debugf("Falling back to the standard CDI modifier")
	}

	return cdi.New(
		cdi.WithLogger(logger),
		cdi.WithDevices(devices...),
		cdi.WithSpecDirs(cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs...),
	)
}

func getDevicesFromSpec(logger logger.Interface, ociSpec oci.Spec, cfg *config.Config) ([]string, error) {
	rawSpec, err := ociSpec.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	annotationDevices, err := getAnnotationDevices(cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.AnnotationPrefixes, rawSpec.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container annotations: %v", err)
	}
	if len(annotationDevices) > 0 {
		return annotationDevices, nil
	}

	container, err := image.NewCUDAImageFromSpec(rawSpec)
	if err != nil {
		return nil, err
	}
	if cfg.AcceptDeviceListAsVolumeMounts {
		mountDevices := container.CDIDevicesFromMounts()
		if len(mountDevices) > 0 {
			return mountDevices, nil
		}
	}

	envDevices := container.DevicesFromEnvvars(visibleDevicesEnvvar)

	var devices []string
	seen := make(map[string]bool)
	for _, name := range envDevices.List() {
		if !parser.IsQualifiedName(name) {
			name = fmt.Sprintf("%s=%s", cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.DefaultKind, name)
		}
		if seen[name] {
			logger.Debugf("Ignoring duplicate device %q", name)
			continue
		}
		devices = append(devices, name)
	}

	if len(devices) == 0 {
		return nil, nil
	}

	if cfg.AcceptEnvvarUnprivileged || image.IsPrivileged(rawSpec) {
		return devices, nil
	}

	logger.Warningf("Ignoring devices specified in NVIDIA_VISIBLE_DEVICES: %v", devices)

	return nil, nil
}

// getAnnotationDevices returns a list of devices specified in the annotations.
// Keys starting with the specified prefixes are considered and expected to contain a comma-separated list of
// fully-qualified CDI devices names. If any device name is not fully-quality an error is returned.
// The list of returned devices is deduplicated.
func getAnnotationDevices(prefixes []string, annotations map[string]string) ([]string, error) {
	devicesByKey := make(map[string][]string)
	for key, value := range annotations {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				devicesByKey[key] = strings.Split(value, ",")
			}
		}
	}

	seen := make(map[string]bool)
	var annotationDevices []string
	for key, devices := range devicesByKey {
		for _, device := range devices {
			if !parser.IsQualifiedName(device) {
				return nil, fmt.Errorf("invalid device name %q in annotation %q", device, key)
			}
			if seen[device] {
				continue
			}
			annotationDevices = append(annotationDevices, device)
			seen[device] = true
		}
	}

	return annotationDevices, nil
}

// filterAutomaticDevices searches for "automatic" device names in the input slice.
// "Automatic" devices are a well-defined list of CDI device names which, when requested,
// trigger the generation of a CDI spec at runtime. This removes the need to generate a
// CDI spec on the system a-priori as well as keep it up-to-date.
func filterAutomaticDevices(devices []string) []string {
	var automatic []string
	for _, device := range devices {
		vendor, class, _ := parser.ParseDevice(device)
		if vendor == "runtime.nvidia.com" && class == "gpu" {
			automatic = append(automatic, device)
		}
	}
	return automatic
}

func newAutomaticCDISpecModifier(logger logger.Interface, cfg *config.Config, devices []string) (oci.SpecModifier, error) {
	logger.Debugf("Generating in-memory CDI specs for devices %v", devices)
	spec, err := generateAutomaticCDISpec(logger, cfg, devices)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI spec: %w", err)
	}
	cdiModifier, err := cdi.New(
		cdi.WithLogger(logger),
		cdi.WithSpec(spec.Raw()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct CDI modifier: %w", err)
	}

	return cdiModifier, nil
}

func generateAutomaticCDISpec(logger logger.Interface, cfg *config.Config, devices []string) (spec.Interface, error) {
	cdilib, err := nvcdi.New(
		nvcdi.WithLogger(logger),
		nvcdi.WithNVIDIACTKPath(cfg.NVIDIACTKConfig.Path),
		nvcdi.WithDriverRoot(cfg.NVIDIAContainerCLIConfig.Root),
		nvcdi.WithVendor("runtime.nvidia.com"),
		nvcdi.WithClass("gpu"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct CDI library: %w", err)
	}

	identifiers := []string{}
	for _, device := range devices {
		_, _, id := parser.ParseDevice(device)
		identifiers = append(identifiers, id)
	}

	deviceSpecs, err := cdilib.GetDeviceSpecsByID(identifiers...)
	if err != nil {
		return nil, fmt.Errorf("failed to get CDI device specs: %w", err)
	}

	commonEdits, err := cdilib.GetCommonEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to get common CDI spec edits: %w", err)
	}

	return spec.New(
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*commonEdits.ContainerEdits),
		spec.WithVendor("runtime.nvidia.com"),
		spec.WithClass("gpu"),
	)
}
