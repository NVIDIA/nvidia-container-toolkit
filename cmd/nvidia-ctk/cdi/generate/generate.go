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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	cdi "tags.cncf.io/container-device-interface/pkg/parser"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

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

	config *configAsValueSource
}

type options struct {
	output               string
	format               string
	deviceNameStrategies []string
	driverRoot           string
	devRoot              string
	nvidiaCDIHookPath    string
	ldconfigPath         string
	mode                 string
	vendor               string
	class                string

	configSearchPaths  []string
	librarySearchPaths []string
	disabledHooks      []string
	enabledHooks       []string

	featureFlags []string

	csv struct {
		files          []string
		ignorePatterns []string
	}

	noAllDevice bool
	deviceIDs   []string

	// the following are used for dependency injection during spec generation.
	nvmllib nvml.Interface
}

// NewCommand constructs a generate-cdi command with the specified logger
func NewCommand(logger logger.Interface, configFilePath *string) *cli.Command {
	c := command{
		logger: logger,
		config: New(configFilePath),
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	opts := options{}

	// Create the 'generate-cdi' command
	c := cli.Command{
		Name:                   "generate",
		Usage:                  "Generate CDI specifications for use with CDI-enabled runtimes",
		UseShortOptionHandling: true,
		EnableShellCompletion:  true,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(cmd, &opts)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.run(&opts)
		},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "config-search-path",
				Usage:       "Specify the path to search for config files when discovering the entities that should be included in the CDI specification.",
				Destination: &opts.configSearchPaths,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_CONFIG_SEARCH_PATHS"),
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "Specify the file to output the generated CDI specification to. If this is '' the specification is output to STDOUT",
				Destination: &opts.output,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_OUTPUT_FILE_PATH"),
			},
			&cli.StringFlag{
				Name:        "format",
				Usage:       "The output format for the generated spec [json | yaml]. This overrides the format defined by the output file extension (if specified).",
				Value:       spec.FormatYAML,
				Destination: &opts.format,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_OUTPUT_FORMAT"),
			},
			&cli.StringFlag{
				Name:    "mode",
				Aliases: []string{"discovery-mode"},
				Usage: "The mode to use when discovering the available entities. " +
					"One of [" + strings.Join(nvcdi.AllModes[string](), " | ") + "]. " +
					"If mode is set to 'auto' the mode will be determined based on the system configuration.",
				Value:       string(nvcdi.ModeAuto),
				Destination: &opts.mode,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("NVIDIA_CTK_CDI_GENERATE_MODE"),
				),
			},
			&cli.StringFlag{
				Name:        "dev-root",
				Usage:       "Specify the root where `/dev` is located. If this is not specified, the driver-root is assumed.",
				Destination: &opts.devRoot,
				Sources:     cli.EnvVars("NVIDIA_CTK_DEV_ROOT"),
			},
			&cli.StringSliceFlag{
				Name:        "device-name-strategy",
				Usage:       "Specify the strategy for generating device names. If this is specified multiple times, the devices will be duplicated for each strategy. One of [index | uuid | type-index]",
				Value:       []string{nvcdi.DeviceNameStrategyIndex, nvcdi.DeviceNameStrategyUUID},
				Destination: &opts.deviceNameStrategies,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_DEVICE_NAME_STRATEGIES"),
			},
			&cli.StringFlag{
				Name:        "driver-root",
				Usage:       "Specify the NVIDIA GPU driver root to use when discovering the entities that should be included in the CDI specification.",
				Destination: &opts.driverRoot,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("NVIDIA_CTK_DRIVER_ROOT"),
					m.config.ValueFrom("nvidia-container-cli.root"),
				),
			},
			&cli.StringSliceFlag{
				Name:        "library-search-path",
				Usage:       "Specify the path to search for libraries when discovering the entities that should be included in the CDI specification.\n\tNote: This option only applies to CSV mode.",
				Destination: &opts.librarySearchPaths,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_LIBRARY_SEARCH_PATHS"),
			},
			&cli.StringFlag{
				Name:    "nvidia-cdi-hook-path",
				Aliases: []string{"nvidia-ctk-path"},
				Usage: "Specify the path to use for the nvidia-cdi-hook in the generated CDI specification. " +
					"If not specified, the PATH will be searched for `nvidia-cdi-hook`. " +
					"NOTE: That if this is specified as `nvidia-ctk`, the PATH will be searched for `nvidia-ctk` instead.",
				Destination: &opts.nvidiaCDIHookPath,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_HOOK_PATH"),
			},
			&cli.StringFlag{
				Name:        "ldconfig-path",
				Usage:       "Specify the path to use for ldconfig in the generated CDI specification",
				Destination: &opts.ldconfigPath,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("NVIDIA_CTK_CDI_GENERATE_LDCONFIG_PATH"),
				),
			},
			&cli.StringFlag{
				Name:        "vendor",
				Aliases:     []string{"cdi-vendor"},
				Usage:       "the vendor string to use for the generated CDI specification.",
				Value:       "nvidia.com",
				Destination: &opts.vendor,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_VENDOR"),
			},
			&cli.StringFlag{
				Name:        "class",
				Aliases:     []string{"cdi-class"},
				Usage:       "the class string to use for the generated CDI specification.",
				Value:       "gpu",
				Destination: &opts.class,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_CLASS"),
			},
			&cli.StringSliceFlag{
				Name:        "csv.file",
				Usage:       "The path to the list of CSV files to use when generating the CDI specification in CSV mode.",
				Value:       csv.DefaultFileList(),
				Destination: &opts.csv.files,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_CSV_FILES"),
			},
			&cli.StringSliceFlag{
				Name:        "csv.ignore-pattern",
				Usage:       "specify a pattern the CSV mount specifications.",
				Destination: &opts.csv.ignorePatterns,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_CSV_IGNORE_PATTERNS"),
			},
			&cli.StringSliceFlag{
				Name:    "disable-hook",
				Aliases: []string{"disable-hooks"},
				Usage: "specify a specific hook to skip when generating CDI " +
					"specifications. This can be specified multiple times and the " +
					"special hook name 'all' can be used ensure that the generated " +
					"CDI specification does not include any hooks.",
				Destination: &opts.disabledHooks,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_DISABLED_HOOKS"),
			},
			&cli.StringSliceFlag{
				Name:        "enable-hook",
				Aliases:     []string{"enable-hooks"},
				Usage:       "Explicitly enable a hook in the generated CDI specification. This overrides disabled hooks. This can be specified multiple times.",
				Destination: &opts.enabledHooks,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_ENABLED_HOOKS"),
			},
			&cli.StringSliceFlag{
				Name:        "feature-flag",
				Aliases:     []string{"feature-flags"},
				Usage:       "specify feature flags for CDI spec generation",
				Destination: &opts.featureFlags,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_FEATURE_FLAGS"),
			},
			&cli.BoolFlag{
				Name:        "no-all-device",
				Usage:       "Don't generate an `all` device for the resultant spec",
				Destination: &opts.noAllDevice,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_NO_ALL_DEVICE"),
			},
			&cli.StringSliceFlag{
				Name:        "device-id",
				Aliases:     []string{"device-ids", "device", "devices"},
				Usage:       "Restrict generation to the specified device identifiers",
				Value:       []string{"all"},
				Destination: &opts.deviceIDs,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_GENERATE_DEVICE_IDS"),
			},
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Command, opts *options) error {
	opts.format = strings.ToLower(opts.format)
	switch opts.format {
	case spec.FormatJSON:
	case spec.FormatYAML:
	default:
		return fmt.Errorf("invalid output format: %v", opts.format)
	}

	opts.mode = strings.ToLower(opts.mode)
	if !nvcdi.IsValidMode(opts.mode) {
		return fmt.Errorf("invalid discovery mode: %v", opts.mode)
	}

	for _, strategy := range opts.deviceNameStrategies {
		_, err := nvcdi.NewDeviceNamer(strategy)
		if err != nil {
			return err
		}
	}

	opts.nvidiaCDIHookPath = config.ResolveNVIDIACDIHookPath(m.logger, opts.nvidiaCDIHookPath)

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

	for _, hook := range opts.enabledHooks {
		if hook == "all" {
			return fmt.Errorf("enabling all hooks is not supported")
		}
	}

	if slices.Contains(opts.deviceIDs, "none") && !opts.noAllDevice {
		m.logger.Warning("Disabling generation of 'all' device")
		opts.noAllDevice = true
	}
	return nil
}

func (m command) run(opts *options) error {
	specs, err := m.generateSpecs(opts)
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}

	var errs error
	for _, spec := range specs {
		m.logger.Infof("Generated CDI spec with version %v", spec.Raw().Version)

		errs = errors.Join(errs, spec.Save(opts.output))
	}
	return errs
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

type generatedSpecs struct {
	spec.Interface
	filenameInfix string
}

func (g *generatedSpecs) Save(filename string) error {
	filename = g.updateFilename(filename)

	if filename == "" {
		_, err := g.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write CDI spec to STDOUT: %v", err)
		}
		return nil
	}

	return g.Interface.Save(filename)
}

func (g generatedSpecs) updateFilename(filename string) string {
	if g.filenameInfix == "" || filename == "" {
		return filename
	}
	ext := filepath.Ext(filepath.Base(filename))
	return strings.TrimSuffix(filename, ext) + g.filenameInfix + ext
}

func (m command) generateSpecs(opts *options) ([]generatedSpecs, error) {
	var deviceNamers []nvcdi.DeviceNamer
	for _, strategy := range opts.deviceNameStrategies {
		deviceNamer, err := nvcdi.NewDeviceNamer(strategy)
		if err != nil {
			return nil, fmt.Errorf("failed to create device namer: %v", err)
		}
		deviceNamers = append(deviceNamers, deviceNamer)
	}

	cdiOptions := []nvcdi.Option{
		nvcdi.WithLogger(m.logger),
		nvcdi.WithDriverRoot(opts.driverRoot),
		nvcdi.WithDevRoot(opts.devRoot),
		nvcdi.WithNVIDIACDIHookPath(opts.nvidiaCDIHookPath),
		nvcdi.WithLdconfigPath(opts.ldconfigPath),
		nvcdi.WithDeviceNamers(deviceNamers...),
		nvcdi.WithMode(opts.mode),
		nvcdi.WithConfigSearchPaths(opts.configSearchPaths),
		nvcdi.WithLibrarySearchPaths(opts.librarySearchPaths),
		nvcdi.WithCSVFiles(opts.csv.files),
		nvcdi.WithCSVIgnorePatterns(opts.csv.ignorePatterns),
		nvcdi.WithDisabledHooks(opts.disabledHooks...),
		nvcdi.WithEnabledHooks(opts.enabledHooks...),
		nvcdi.WithFeatureFlags(opts.featureFlags...),
		// We set the following to allow for dependency injection:
		nvcdi.WithNvmlLib(opts.nvmllib),
	}

	cdilib, err := nvcdi.New(cdiOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDI library: %v", err)
	}

	allDeviceSpecs, err := cdilib.GetDeviceSpecsByID(opts.deviceIDs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create device CDI specs: %v", err)
	}

	commonEdits, err := cdilib.GetCommonEdits()
	if err != nil {
		return nil, fmt.Errorf("failed to create edits common for entities: %v", err)
	}

	commonSpecOptions := []spec.Option{
		spec.WithVendor(opts.vendor),
		spec.WithEdits(*commonEdits.ContainerEdits),
		spec.WithFormat(opts.format),
		spec.WithPermissions(0644),
	}

	if !opts.noAllDevice {
		commonSpecOptions = append(commonSpecOptions,
			spec.WithMergedDeviceOptions(
				transform.WithName(allDeviceName),
				transform.WithSkipIfExists(true),
			),
		)
	}

	fullSpec, err := spec.New(
		append(commonSpecOptions,
			spec.WithClass(opts.class),
			spec.WithDeviceSpecs(allDeviceSpecs),
		)...,
	)
	if err != nil {
		return nil, err
	}
	var allSpecs []generatedSpecs

	allSpecs = append(allSpecs, generatedSpecs{Interface: fullSpec, filenameInfix: ""})

	deviceSpecsByDeviceCoherence := (deviceSpecs)(allDeviceSpecs).splitOnAnnotation("gpu.nvidia.com/coherent")

	if coherentDeviceSpecs := deviceSpecsByDeviceCoherence["gpu.nvidia.com/coherent=true"]; len(coherentDeviceSpecs) > 0 {
		infix := ".coherent"
		coherentSpecs, err := spec.New(
			append(commonSpecOptions,
				spec.WithClass(opts.class+infix),
				spec.WithDeviceSpecs(coherentDeviceSpecs),
			)...,
		)
		if err != nil {
			return nil, err
		}
		allSpecs = append(allSpecs, generatedSpecs{Interface: coherentSpecs, filenameInfix: infix})
	}

	if noncoherentDeviceSpecs := deviceSpecsByDeviceCoherence["gpu.nvidia.com/coherent=false"]; len(noncoherentDeviceSpecs) > 0 {
		infix := ".noncoherent"
		noncoherentSpecs, err := spec.New(
			append(commonSpecOptions,
				spec.WithClass(opts.class+infix),
				spec.WithDeviceSpecs(noncoherentDeviceSpecs),
			)...,
		)

		if err != nil {
			return nil, err
		}
		allSpecs = append(allSpecs, generatedSpecs{Interface: noncoherentSpecs, filenameInfix: infix})
	}

	return allSpecs, nil
}

type deviceSpecs []specs.Device

func (d deviceSpecs) splitOnAnnotation(key string) map[string][]specs.Device {
	splitSpecs := make(map[string][]specs.Device)

	var specsToRemoveAnnotations []*specs.Device
	for _, deviceSpec := range d {
		value, ok := deviceSpec.Annotations[key]
		if !ok {
			continue
		}
		splitSpecs[key+"="+value] = append(splitSpecs[key+"="+value], deviceSpec)
		specsToRemoveAnnotations = append(specsToRemoveAnnotations, &deviceSpec)
	}

	// We also remove the annotations that were used to split the devices:
	for _, deviceSpec := range specsToRemoveAnnotations {
		if _, ok := deviceSpec.Annotations[key]; !ok {
			continue
		}
		delete(deviceSpec.Annotations, key)
	}

	return splitSpecs
}
