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

	"github.com/urfave/cli/v2"
	cdi "tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

const (
	allDeviceName = "all"
)

type command struct {
	logger logger.Interface
}

type options struct {
	output             string
	format             string
	deviceNameStrategy string
	driverRoot         string
	devRoot            string
	nvidiaCTKPath      string
	mode               string
	vendor             string
	class              string

	librarySearchPaths cli.StringSlice

	csv struct {
		files          cli.StringSlice
		ignorePatterns cli.StringSlice
	}
}

// NewCommand constructs a generate-cdi command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	opts := options{}

	// Create the 'generate-cdi' command
	c := cli.Command{
		Name:  "generate",
		Usage: "Generate CDI specifications for use with CDI-enabled runtimes",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &opts)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Usage:       "Specify the file to output the generated CDI specification to. If this is '' the specification is output to STDOUT",
			Destination: &opts.output,
		},
		&cli.StringFlag{
			Name:        "format",
			Usage:       "The output format for the generated spec [json | yaml]. This overrides the format defined by the output file extension (if specified).",
			Value:       spec.FormatYAML,
			Destination: &opts.format,
		},
		&cli.StringFlag{
			Name:        "mode",
			Aliases:     []string{"discovery-mode"},
			Usage:       "The mode to use when discovering the available entities. One of [auto | nvml | wsl]. If mode is set to 'auto' the mode will be determined based on the system configuration.",
			Value:       nvcdi.ModeAuto,
			Destination: &opts.mode,
		},
		&cli.StringFlag{
			Name:        "dev-root",
			Usage:       "Specify the root where `/dev` is located. If this is not specified, the driver-root is assumed.",
			Destination: &opts.devRoot,
		},
		&cli.StringFlag{
			Name:        "device-name-strategy",
			Usage:       "Specify the strategy for generating device names. One of [index | uuid | type-index]",
			Value:       nvcdi.DeviceNameStrategyIndex,
			Destination: &opts.deviceNameStrategy,
		},
		&cli.StringFlag{
			Name:        "driver-root",
			Usage:       "Specify the NVIDIA GPU driver root to use when discovering the entities that should be included in the CDI specification.",
			Destination: &opts.driverRoot,
		},
		&cli.StringSliceFlag{
			Name:        "library-search-path",
			Usage:       "Specify the path to search for libraries when discovering the entities that should be included in the CDI specification.\n\tNote: This option only applies to CSV mode.",
			Destination: &opts.librarySearchPaths,
		},
		&cli.StringFlag{
			Name:        "nvidia-ctk-path",
			Usage:       "Specify the path to use for the nvidia-ctk in the generated CDI specification. If this is left empty, the path will be searched.",
			Destination: &opts.nvidiaCTKPath,
		},
		&cli.StringFlag{
			Name:        "vendor",
			Aliases:     []string{"cdi-vendor"},
			Usage:       "the vendor string to use for the generated CDI specification.",
			Value:       "nvidia.com",
			Destination: &opts.vendor,
		},
		&cli.StringFlag{
			Name:        "class",
			Aliases:     []string{"cdi-class"},
			Usage:       "the class string to use for the generated CDI specification.",
			Value:       "gpu",
			Destination: &opts.class,
		},
		&cli.StringSliceFlag{
			Name:        "csv.file",
			Usage:       "The path to the list of CSV files to use when generating the CDI specification in CSV mode.",
			Value:       cli.NewStringSlice(csv.DefaultFileList()...),
			Destination: &opts.csv.files,
		},
		&cli.StringSliceFlag{
			Name:        "csv.ignore-pattern",
			Usage:       "Specify a pattern the CSV mount specifications.",
			Destination: &opts.csv.ignorePatterns,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	opts.format = strings.ToLower(opts.format)
	switch opts.format {
	case spec.FormatJSON:
	case spec.FormatYAML:
	default:
		return fmt.Errorf("invalid output format: %v", opts.format)
	}

	opts.mode = strings.ToLower(opts.mode)
	switch opts.mode {
	case nvcdi.ModeAuto:
	case nvcdi.ModeCSV:
	case nvcdi.ModeNvml:
	case nvcdi.ModeWsl:
	case nvcdi.ModeManagement:
	default:
		return fmt.Errorf("invalid discovery mode: %v", opts.mode)
	}

	_, err := nvcdi.NewDeviceNamer(opts.deviceNameStrategy)
	if err != nil {
		return err
	}

	opts.nvidiaCTKPath = config.ResolveNVIDIACTKPath(m.logger, opts.nvidiaCTKPath)

	if outputFileFormat := formatFromFilename(opts.output); outputFileFormat != "" {
		m.logger.Debugf("Inferred output format as %q from output file name", outputFileFormat)
		if !c.IsSet("format") {
			opts.format = outputFileFormat
		} else if outputFileFormat != opts.format {
			m.logger.Warningf("Requested output format %q does not match format implied by output file name: %q", opts.format, outputFileFormat)
		}
	}

	if err := cdi.ValidateVendorName(opts.vendor); err != nil {
		return fmt.Errorf("invalid CDI vendor name: %v", err)
	}
	if err := cdi.ValidateClassName(opts.class); err != nil {
		return fmt.Errorf("invalid CDI class name: %v", err)
	}
	return nil
}

func (m command) run(c *cli.Context, opts *options) error {
	spec, err := m.generateSpec(opts)
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}
	m.logger.Infof("Generated CDI spec with version %v", spec.Raw().Version)

	if opts.output == "" {
		_, err := spec.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	return spec.Save(opts.output)
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

func (m command) generateSpec(opts *options) (spec.Interface, error) {
	deviceNamer, err := nvcdi.NewDeviceNamer(opts.deviceNameStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create device namer: %v", err)
	}

	cdilib, err := nvcdi.New(
		nvcdi.WithLogger(m.logger),
		nvcdi.WithDriverRoot(opts.driverRoot),
		nvcdi.WithDevRoot(opts.devRoot),
		nvcdi.WithNVIDIACTKPath(opts.nvidiaCTKPath),
		nvcdi.WithDeviceNamer(deviceNamer),
		nvcdi.WithMode(opts.mode),
		nvcdi.WithLibrarySearchPaths(opts.librarySearchPaths.Value()),
		nvcdi.WithCSVFiles(opts.csv.files.Value()),
		nvcdi.WithCSVIgnorePatterns(opts.csv.ignorePatterns.Value()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDI library: %v", err)
	}

	deviceSpecs, err := cdilib.GetAllDeviceSpecs()
	if err != nil {
		return nil, fmt.Errorf("failed to create device CDI specs: %v", err)
	}

	commonEdits, err := cdilib.GetCommonEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to create edits common for entities: %v", err)
	}

	return spec.New(
		spec.WithVendor(opts.vendor),
		spec.WithClass(opts.class),
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*commonEdits.ContainerEdits),
		spec.WithFormat(opts.format),
		spec.WithMergedDeviceOptions(
			transform.WithName(allDeviceName),
			transform.WithSkipIfExists(true),
		),
		spec.WithPermissions(0644),
	)
}
