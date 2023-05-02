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

package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	specs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	allDeviceName = "all"
)

type command struct {
	logger *logrus.Logger
}

type config struct {
	output             string
	format             string
	deviceNameStrategy string
	driverRoot         string
	nvidiaCTKPath      string
	mode               string
	vendor             string
	class              string
}

// NewCommand constructs a generate-cdi command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	cfg := config{}

	// Create the 'generate-cdi' command
	c := cli.Command{
		Name:  "generate",
		Usage: "Generate CDI specifications for use with CDI-enabled runtimes",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Usage:       "Specify the file to output the generated CDI specification to. If this is '' the specification is output to STDOUT",
			Destination: &cfg.output,
		},
		&cli.StringFlag{
			Name:        "format",
			Usage:       "The output format for the generated spec [json | yaml]. This overrides the format defined by the output file extension (if specified).",
			Value:       spec.FormatYAML,
			Destination: &cfg.format,
		},
		&cli.StringFlag{
			Name:        "mode",
			Aliases:     []string{"discovery-mode"},
			Usage:       "The mode to use when discovering the available entities. One of [auto | nvml | wsl]. If mode is set to 'auto' the mode will be determined based on the system configuration.",
			Value:       nvcdi.ModeAuto,
			Destination: &cfg.mode,
		},
		&cli.StringFlag{
			Name:        "device-name-strategy",
			Usage:       "Specify the strategy for generating device names. One of [index | uuid | type-index]",
			Value:       nvcdi.DeviceNameStrategyIndex,
			Destination: &cfg.deviceNameStrategy,
		},
		&cli.StringFlag{
			Name:        "driver-root",
			Usage:       "Specify the NVIDIA GPU driver root to use when discovering the entities that should be included in the CDI specification.",
			Destination: &cfg.driverRoot,
		},
		&cli.StringFlag{
			Name:        "nvidia-ctk-path",
			Usage:       "Specify the path to use for the nvidia-ctk in the generated CDI specification. If this is left empty, the path will be searched.",
			Destination: &cfg.nvidiaCTKPath,
		},
		&cli.StringFlag{
			Name:        "vendor",
			Aliases:     []string{"cdi-vendor"},
			Usage:       "the vendor string to use for the generated CDI specification.",
			Value:       "nvidia.com",
			Destination: &cfg.vendor,
		},
		&cli.StringFlag{
			Name:        "class",
			Aliases:     []string{"cdi-class"},
			Usage:       "the class string to use for the generated CDI specification.",
			Value:       "gpu",
			Destination: &cfg.class,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, cfg *config) error {

	cfg.format = strings.ToLower(cfg.format)
	switch cfg.format {
	case spec.FormatJSON:
	case spec.FormatYAML:
	default:
		return fmt.Errorf("invalid output format: %v", cfg.format)
	}

	cfg.mode = strings.ToLower(cfg.mode)
	switch cfg.mode {
	case nvcdi.ModeAuto:
	case nvcdi.ModeNvml:
	case nvcdi.ModeWsl:
	case nvcdi.ModeManagement:
	default:
		return fmt.Errorf("invalid discovery mode: %v", cfg.mode)
	}

	_, err := nvcdi.NewDeviceNamer(cfg.deviceNameStrategy)
	if err != nil {
		return err
	}

	cfg.nvidiaCTKPath = discover.FindNvidiaCTK(m.logger, cfg.nvidiaCTKPath)

	if outputFileFormat := formatFromFilename(cfg.output); outputFileFormat != "" {
		m.logger.Debugf("Inferred output format as %q from output file name", outputFileFormat)
		if !c.IsSet("format") {
			cfg.format = outputFileFormat
		} else if outputFileFormat != cfg.format {
			m.logger.Warningf("Requested output format %q does not match format implied by output file name: %q", cfg.format, outputFileFormat)
		}
	}

	if err := cdi.ValidateVendorName(cfg.vendor); err != nil {
		return fmt.Errorf("invalid CDI vendor name: %v", err)
	}
	if err := cdi.ValidateClassName(cfg.class); err != nil {
		return fmt.Errorf("invalid CDI class name: %v", err)
	}
	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.generateSpec(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}
	m.logger.Infof("Generated CDI spec with version %v", spec.Raw().Version)

	if cfg.output == "" {
		_, err := spec.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	return spec.Save(cfg.output)
}

func formatFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".json":
		return spec.FormatJSON
	case ".yaml", ".yml":
		return spec.FormatYAML
	}

	return ""
}

func (m command) generateSpec(cfg *config) (spec.Interface, error) {
	deviceNamer, err := nvcdi.NewDeviceNamer(cfg.deviceNameStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create device namer: %v", err)
	}

	cdilib, err := nvcdi.New(
		nvcdi.WithLogger(m.logger),
		nvcdi.WithDriverRoot(cfg.driverRoot),
		nvcdi.WithNVIDIACTKPath(cfg.nvidiaCTKPath),
		nvcdi.WithDeviceNamer(deviceNamer),
		nvcdi.WithMode(string(cfg.mode)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDI library: %v", err)
	}

	deviceSpecs, err := cdilib.GetAllDeviceSpecs()
	if err != nil {
		return nil, fmt.Errorf("failed to create device CDI specs: %v", err)
	}
	var hasAll bool
	for _, deviceSpec := range deviceSpecs {
		if deviceSpec.Name == allDeviceName {
			hasAll = true
			break
		}
	}
	if !hasAll {
		allDevice, err := MergeDeviceSpecs(deviceSpecs, allDeviceName)
		if err != nil {
			return nil, fmt.Errorf("failed to create CDI specification for %q device: %v", allDeviceName, err)
		}
		deviceSpecs = append(deviceSpecs, allDevice)
	}

	commonEdits, err := cdilib.GetCommonEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to create edits common for entities: %v", err)
	}

	return spec.New(
		spec.WithVendor(cfg.vendor),
		spec.WithClass(cfg.class),
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*commonEdits.ContainerEdits),
		spec.WithFormat(cfg.format),
		spec.WithPermissions(0644),
	)
}

// MergeDeviceSpecs creates a device with the specified name which combines the edits from the previous devices.
// If a device of the specified name already exists, an error is returned.
func MergeDeviceSpecs(deviceSpecs []specs.Device, mergedDeviceName string) (specs.Device, error) {
	if err := cdi.ValidateDeviceName(mergedDeviceName); err != nil {
		return specs.Device{}, fmt.Errorf("invalid device name %q: %v", mergedDeviceName, err)
	}
	for _, d := range deviceSpecs {
		if d.Name == mergedDeviceName {
			return specs.Device{}, fmt.Errorf("device %q already exists", mergedDeviceName)
		}
	}

	mergedEdits := edits.NewContainerEdits()

	for _, d := range deviceSpecs {
		edit := cdi.ContainerEdits{
			ContainerEdits: &d.ContainerEdits,
		}
		mergedEdits.Append(&edit)
	}

	merged := specs.Device{
		Name:           mergedDeviceName,
		ContainerEdits: *mergedEdits.ContainerEdits,
	}
	return merged, nil
}
