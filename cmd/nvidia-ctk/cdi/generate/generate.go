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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	specs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
	"sigs.k8s.io/yaml"
)

const (
	formatJSON = "json"
	formatYAML = "yaml"
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

func (m command) validateFlags(r *cli.Context, cfg *config) error {
	cfg.format = strings.ToLower(cfg.format)
	switch cfg.format {
	case formatJSON:
	case formatYAML:
	default:
		return fmt.Errorf("invalid output format: %v", cfg.format)
	}

	_, err := nvcdi.NewDeviceNamer(cfg.deviceNameStrategy)
	if err != nil {
		return err
	}

	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.generateSpec(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}

	var outputTo io.Writer
	if cfg.output == "" {
		outputTo = os.Stdout
	} else {
		err := createParentDirsIfRequired(cfg.output)
		if err != nil {
			return fmt.Errorf("failed to create parent folders for output file: %v", err)
		}
		outputFile, err := os.Create(cfg.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outputFile.Close()
		outputTo = outputFile
	}

	if outputFileFormat := formatFromFilename(cfg.output); outputFileFormat != "" {
		m.logger.Debugf("Inferred output format as %q from output file name", outputFileFormat)
		if !c.IsSet("format") {
			cfg.format = outputFileFormat
		} else if outputFileFormat != cfg.format {
			m.logger.Warningf("Requested output format %q does not match format implied by output file name: %q", cfg.format, outputFileFormat)
		}
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal CDI spec: %v", err)
	}

	if strings.ToLower(cfg.format) == formatJSON {
		data, err = yaml.YAMLToJSONStrict(data)
		if err != nil {
			return fmt.Errorf("failed to convert CDI spec from YAML to JSON: %v", err)
		}
	}

	err = writeToOutput(cfg.format, data, outputTo)
	if err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	return nil
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

func writeToOutput(format string, data []byte, output io.Writer) error {
	if format == formatYAML {
		_, err := output.Write([]byte("---\n"))
		if err != nil {
			return fmt.Errorf("failed to write YAML separator: %v", err)
		}
	}
	_, err := output.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %v", err)
	}

	return nil
}

func (m command) generateSpec(cfg *config) (*specs.Spec, error) {
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
	)

	deviceSpecs, err := cdilib.GetAllDeviceSpecs()
	if err != nil {
		return nil, fmt.Errorf("failed to create device CDI specs: %v", err)
	}

	allDevice := createAllDevice(deviceSpecs)

	deviceSpecs = append(deviceSpecs, allDevice)

	commonEdits, err := cdilib.GetCommonEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to create edits common for entities: %v", err)
	}
	deviceFolderPermissionEdits, err := GetDeviceFolderPermissionHookEdits(m.logger, cfg.driverRoot, cfg.nvidiaCTKPath, deviceSpecs)
	if err != nil {
		return nil, fmt.Errorf("failed to generated edits for device folder permissions: %v", err)
	}

	commonEdits.Append(deviceFolderPermissionEdits)

	// We construct the spec and determine the minimum required version based on the specification.
	spec := specs.Spec{
		Version:        "NOT_SET",
		Kind:           "nvidia.com/gpu",
		Devices:        deviceSpecs,
		ContainerEdits: *commonEdits.ContainerEdits,
	}

	minVersion, err := cdi.MinimumRequiredVersion(&spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get minumum required CDI spec version: %v", err)
	}
	m.logger.Infof("Using minimum required CDI spec version: %s", minVersion)

	spec.Version = minVersion

	return &spec, nil
}

// createAllDevice creates an 'all' device which combines the edits from the previous devices
func createAllDevice(deviceSpecs []specs.Device) specs.Device {
	edits := edits.NewContainerEdits()

	for _, d := range deviceSpecs {
		edit := cdi.ContainerEdits{
			ContainerEdits: &d.ContainerEdits,
		}
		edits.Append(&edit)
	}

	all := specs.Device{
		Name:           "all",
		ContainerEdits: *edits.ContainerEdits,
	}
	return all
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
