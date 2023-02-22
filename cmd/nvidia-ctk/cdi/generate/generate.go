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
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

const (
	formatJSON = "json"
	formatYAML = "yaml"

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
	discoveryMode      string
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
			Value:       formatYAML,
			Destination: &cfg.format,
		},
		&cli.StringFlag{
			Name:        "discovery-mode",
			Usage:       "The mode to use when discovering the available entities. One of [auto | nvml | wsl]. If mode is set to 'auto' the mode will be determined based on the system configuration.",
			Value:       nvcdi.ModeAuto,
			Destination: &cfg.discoveryMode,
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
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, cfg *config) error {

	cfg.format = strings.ToLower(cfg.format)
	switch cfg.format {
	case formatJSON:
	case formatYAML:
	default:
		return fmt.Errorf("invalid output format: %v", cfg.format)
	}

	cfg.discoveryMode = strings.ToLower(cfg.discoveryMode)
	switch cfg.discoveryMode {
	case nvcdi.ModeAuto:
	case nvcdi.ModeNvml:
	case nvcdi.ModeWsl:
	default:
		return fmt.Errorf("invalid discovery mode: %v", cfg.discoveryMode)
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

	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.generateSpec(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}
	m.logger.Infof("Generated CDI spec with version", spec.Raw().Version)

	if cfg.output == "" {
		_, err := spec.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	err = createParentDirsIfRequired(cfg.output)
	if err != nil {
		return fmt.Errorf("failed to create parent folders for output file: %v", err)
	}
	return spec.Save(cfg.output)
}

func formatFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".json":
		return formatJSON
	case ".yaml":
		return formatYAML
	case ".yml":
		return formatYAML
	}

	return ""
}

func (m command) generateSpec(cfg *config) (spec.Interface, error) {
	deviceNamer, err := nvcdi.NewDeviceNamer(cfg.deviceNameStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create device namer: %v", err)
	}

	nvmllib := nvml.New()
	if r := nvmllib.Init(); r != nvml.SUCCESS {
		return nil, r
	}
	defer nvmllib.Shutdown()

	devicelib := device.New(device.WithNvml(nvmllib))

	cdilib := nvcdi.New(
		nvcdi.WithLogger(m.logger),
		nvcdi.WithDriverRoot(cfg.driverRoot),
		nvcdi.WithNVIDIACTKPath(cfg.nvidiaCTKPath),
		nvcdi.WithDeviceNamer(deviceNamer),
		nvcdi.WithDeviceLib(devicelib),
		nvcdi.WithNvmlLib(nvmllib),
		nvcdi.WithMode(string(cfg.discoveryMode)),
	)

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
		spec.WithVendor("nvidia.com"),
		spec.WithClass("gpu"),
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*commonEdits.ContainerEdits),
		spec.WithFormat(cfg.format),
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

// createParentDirsIfRequired creates the parent folders of the specified path if requried.
// Note that MkdirAll does not specifically check whether the specified path is non-empty and raises an error if it is.
// The path will be empty if filename in the current folder is specified, for example
func createParentDirsIfRequired(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
