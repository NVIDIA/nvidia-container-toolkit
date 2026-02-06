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

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier/cdi"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
)

const (
	automaticDeviceVendor = "runtime.nvidia.com"
	automaticDeviceClass  = "gpu"
	automaticDeviceKind   = automaticDeviceVendor + "/" + automaticDeviceClass
	automaticDevicePrefix = automaticDeviceKind + "="
)

// NewCDIModifier creates an OCI spec modifier that determines the modifications to make based on the
// CDI specifications available on the system. The NVIDIA_VISIBLE_DEVICES environment variable is
// used to select the devices to include.
func (f *Factory) NewCDIModifier(isJitCDI bool) (oci.SpecModifier, error) {
	defaultKind := f.cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.DefaultKind
	if isJitCDI {
		defaultKind = automaticDeviceKind
	}
	deviceRequestor := newCDIDeviceRequestor(
		f.logger,
		f.image,
		defaultKind,
	)
	devices := deviceRequestor.DeviceRequests()
	if len(devices) == 0 {
		f.logger.Debugf("No devices requested; no modification required.")
		return nil, nil
	}

	automaticDevices := filterAutomaticDevices(devices)
	if len(automaticDevices) != len(devices) && len(automaticDevices) > 0 {
		return nil, fmt.Errorf("requesting a CDI device with vendor 'runtime.nvidia.com' is not supported when requesting other CDI devices")
	}
	if len(automaticDevices) > 0 {
		f.logger.Debugf("Using automatic CDI modfier for devices %v", automaticDevices)
		modifier, err := newJitCDIModifier(f.logger, f.cfg, f.image, automaticDevices)
		if err != nil {
			return nil, fmt.Errorf("failed to create the automatic CDI modifier: %w", err)
		}
		return modifier, nil
	}

	f.logger.Debugf("Creating CDI modifier for devices: %v", devices)
	return cdi.New(
		cdi.WithLogger(f.logger),
		cdi.WithDevices(devices...),
		cdi.WithSpecDirs(f.cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs...),
	)
}

// newJitCDIModifier creates a modifier that for a generated in-memory CDI spec for the specified CDI devices.
func newJitCDIModifier(logger logger.Interface, cfg *config.Config, image *image.CUDA, automaticDevices []string) (oci.SpecModifier, error) {
	if image != nil {
		automaticDevices = append(automaticDevices, withUniqueDevices(gatedDevices(*image)).DeviceRequests()...)
		automaticDevices = append(automaticDevices, withUniqueDevices(imexDevices(*image)).DeviceRequests()...)
	}
	return newAutomaticCDISpecModifier(logger, cfg, automaticDevices)
}

type deviceRequestor interface {
	DeviceRequests() []string
}

type cdiDeviceRequestor struct {
	logger      logger.Interface
	image       *image.CUDA
	defaultKind string
}

func newCDIDeviceRequestor(logger logger.Interface, image *image.CUDA, defaultKind string) deviceRequestor {
	c := &cdiDeviceRequestor{
		logger:      logger,
		image:       image,
		defaultKind: defaultKind,
	}
	return withUniqueDevices(c)
}

func (c *cdiDeviceRequestor) DeviceRequests() []string {
	if c == nil {
		return nil
	}
	var devices []string
	for _, name := range c.image.VisibleDevices() {
		if name == "" {
			name = "none"
		}
		if !parser.IsQualifiedName(name) {
			name = fmt.Sprintf("%s=%s", c.defaultKind, name)
		}
		devices = append(devices, name)
	}

	return devices
}

type gatedDevices image.CUDA

// DeviceRequests returns a list of devices that are required for gated devices.
func (g gatedDevices) DeviceRequests() []string {
	i := (image.CUDA)(g)

	var devices []string
	if i.Getenv("NVIDIA_GDS") == "enabled" {
		devices = append(devices, "mode=gds")
	}
	if i.Getenv("NVIDIA_MOFED") == "enabled" {
		devices = append(devices, "mode=mofed")
	}
	if i.Getenv("NVIDIA_GDRCOPY") == "enabled" {
		devices = append(devices, "mode=gdrcopy")
	}
	if i.Getenv("NVIDIA_NVSWITCH") == "enabled" {
		devices = append(devices, "mode=nvswitch")
	}

	return devices
}

type imexDevices image.CUDA

func (d imexDevices) DeviceRequests() []string {
	var devices []string
	i := (image.CUDA)(d)
	for _, channelID := range i.ImexChannelRequests() {
		devices = append(devices, "mode=imex,id="+channelID)
	}
	return devices
}

// filterAutomaticDevices searches for "automatic" device names in the input slice.
// "Automatic" devices are a well-defined list of CDI device names which, when requested,
// trigger the generation of a CDI spec at runtime. This removes the need to generate a
// CDI spec on the system a-priori as well as keep it up-to-date.
func filterAutomaticDevices(devices []string) []string {
	var automatic []string
	for _, device := range devices {
		if !strings.HasPrefix(device, automaticDevicePrefix) {
			continue
		}
		automatic = append(automatic, device)
	}
	return automatic
}

func newAutomaticCDISpecModifier(logger logger.Interface, cfg *config.Config, devices []string) (oci.SpecModifier, error) {
	logger.Debugf("Generating in-memory CDI specs for devices %v", devices)

	cdiModeIdentifiers := cdiModeIdentfiersFromDevices(devices...)

	nvcdiFeatureFlags := cfg.NVIDIAContainerRuntimeConfig.Modes.JitCDI.NVCDIFeatureFlags
	if cfg.Features.NoAdditionalGIDsForDeviceNodes.IsEnabled() {
		nvcdiFeatureFlags = append(nvcdiFeatureFlags, nvcdi.FeatureDisableMultipleCSVDevices)
	}

	logger.Debugf("Per-mode identifiers: %v", cdiModeIdentifiers)
	var modifiers oci.SpecModifiers
	for _, mode := range cdiModeIdentifiers.modes {
		cdilib, err := nvcdi.New(
			nvcdi.WithLogger(logger),
			nvcdi.WithNVIDIACDIHookPath(cfg.NVIDIACTKConfig.Path),
			nvcdi.WithDriverRoot(cfg.NVIDIAContainerCLIConfig.Root),
			nvcdi.WithVendor(automaticDeviceVendor),
			nvcdi.WithClass(cdiModeIdentifiers.deviceClassByMode[mode]),
			nvcdi.WithMode(mode),
			nvcdi.WithFeatureFlags(cfg.NVIDIAContainerRuntimeConfig.Modes.JitCDI.NVCDIFeatureFlags...),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct CDI library for mode %q: %w", mode, err)
		}

		spec, err := cdilib.GetSpec(cdiModeIdentifiers.idsByMode[mode]...)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CDI spec for mode %q: %w", mode, err)
		}

		cdiDeviceRequestor, err := cdi.New(
			cdi.WithLogger(logger),
			cdi.WithSpec(spec.Raw()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct CDI modifier for mode %q: %w", mode, err)
		}

		modifiers = append(modifiers, cdiDeviceRequestor)
	}

	return modifiers, nil
}

type cdiModeIdentifiers struct {
	modes             []string
	idsByMode         map[string][]string
	deviceClassByMode map[string]string
}

func cdiModeIdentfiersFromDevices(devices ...string) *cdiModeIdentifiers {
	perModeIdentifiers := make(map[string][]string)
	perModeDeviceClass := map[string]string{"auto": automaticDeviceClass}
	var uniqueModes []string
	seen := make(map[string]bool)
	for _, device := range devices {
		mode, id := getModeIdentifier(device)
		if !seen[mode] {
			uniqueModes = append(uniqueModes, mode)
			seen[mode] = true
		}
		if id != "" {
			perModeIdentifiers[mode] = append(perModeIdentifiers[mode], id)
		}
	}

	return &cdiModeIdentifiers{
		modes:             uniqueModes,
		idsByMode:         perModeIdentifiers,
		deviceClassByMode: perModeDeviceClass,
	}
}

func getModeIdentifier(device string) (string, string) {
	if !strings.HasPrefix(device, "mode=") {
		return "auto", strings.TrimPrefix(device, automaticDevicePrefix)
	}
	parts := strings.SplitN(device, ",", 2)
	mode := strings.TrimPrefix(parts[0], "mode=")
	if len(parts) == 2 {
		return mode, strings.TrimPrefix(parts[1], "id=")
	}
	return mode, ""
}

type deduplicatedDeviceRequestor struct {
	deviceRequestor
}

func withUniqueDevices(deviceRequestor deviceRequestor) deviceRequestor {
	return &deduplicatedDeviceRequestor{deviceRequestor: deviceRequestor}
}

func (d *deduplicatedDeviceRequestor) DeviceRequests() []string {
	if d == nil {
		return nil
	}
	seen := make(map[string]bool)
	var devices []string
	for _, device := range d.deviceRequestor.DeviceRequests() {
		if seen[device] {
			continue
		}
		seen[device] = true
		devices = append(devices, device)
	}
	return devices
}
