/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package apply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v3"
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// command represents the apply command for the cdi feature.
type command struct {
	logger logger.Interface
}

type options struct {
	mode        string
	devices     []string
	cdiSpecDirs []string
	output      string
	input       string
}

// NewCommand constructs the apply-cdi command with the specified logger.
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command for apply-cdi.
func (m command) build() *cli.Command {
	opts := options{}

	return &cli.Command{
		Name:                   "apply",
		Usage:                  "Apply CDI specification to different inputs (e.g., containers, configs)",
		UseShortOptionHandling: true,
		EnableShellCompletion:  true,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, m.validateFlags(&opts)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return m.run(ctx, cmd, &opts)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Usage:       "Mode for applying the CDI spec to different inputs.",
				Value:       "oci",
				Destination: &opts.mode,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_APPLY_MODE"),
			},
			&cli.StringSliceFlag{
				Name:        "device",
				Aliases:     []string{"devices"},
				Usage:       "Specify the CDI device names to apply. Device names should be in the format vendor/class=name (e.g., nvidia.com/gpu=0)",
				Destination: &opts.devices,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_APPLY_DEVICES"),
			},
			&cli.StringSliceFlag{
				Name:        "spec-dir",
				Usage:       "specify the directories to scan for CDI specifications",
				Value:       cdi.DefaultSpecDirs,
				Destination: &opts.cdiSpecDirs,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_SPEC_DIRS"),
			},
			&cli.StringFlag{
				Name:        "input",
				Usage:       "Specify the file to read the input OCI spec from. If empty or '-', input is read from stdin. (Only used in 'oci' mode)",
				Destination: &opts.input,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_APPLY_INPUT"),
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "Specify the file to write the output to. If empty or '-', output is written to stdout",
				Destination: &opts.output,
				Sources:     cli.EnvVars("NVIDIA_CTK_CDI_APPLY_OUTPUT"),
			},
		},
	}
}

// validateFlags validates the command line flags.
func (m command) validateFlags(opts *options) error {
	if len(opts.cdiSpecDirs) == 0 {
		return errors.New("at least one CDI specification directory must be specified")
	}
	return nil
}

// run is the action handler for the apply command.
func (m command) run(ctx context.Context, cmd *cli.Command, opts *options) error {
	m.logger.Infof("apply command invoked with mode: %s", opts.mode)

	// If no devices specified, return early
	if len(opts.devices) == 0 {
		m.logger.Infof("No devices specified")
		return nil
	}

	m.logger.Infof("Processing %d CDI device(s)", len(opts.devices))
	m.logger.Debugf("CDI spec directories: %v", opts.cdiSpecDirs)

	// Create a CDI cache/registry
	registry, err := cdi.NewCache(
		cdi.WithAutoRefresh(false),
		cdi.WithSpecDirs(opts.cdiSpecDirs...),
	)
	if err != nil {
		return fmt.Errorf("failed to create CDI cache: %v", err)
	}

	// Refresh the cache to load the CDI specifications
	_ = registry.Refresh()
	if errs := registry.GetErrors(); len(errs) > 0 {
		m.logger.Warningf("The following CDI registry errors were reported:")
		for specFile, err := range errs {
			m.logger.Warningf("%v: %v", specFile, err)
		}
	}

	// Determine output writer
	outputWriter, err := m.getOutputWriter(opts.output)
	if err != nil {
		return fmt.Errorf("failed to create output writer: %v", err)
	}
	defer func() {
		_ = outputWriter.Close()
	}()

	// Apply the device specs based on the selected mode
	switch opts.mode {
	case "fstab":
		if err := m.applyFstabMode(ctx, opts, registry, outputWriter); err != nil {
			return fmt.Errorf("failed to apply fstab mode: %v", err)
		}
	case "oci":
		if err := m.applyOCIMode(ctx, opts, registry, outputWriter); err != nil {
			return fmt.Errorf("failed to apply OCI mode: %v", err)
		}
	default:
		return fmt.Errorf("unsupported mode: %s", opts.mode)
	}

	return nil
}

// applyFstabMode generates fstab entries for all devices and writes them to the output.
func (m command) applyFstabMode(ctx context.Context, opts *options, registry *cdi.Cache, outputWriter io.Writer) error {

	edits := &ContainerEdits{
		ContainerEdits: edits.NewContainerEdits(),
	}
	specs := map[*cdi.Spec]struct{}{}

	var unresolved []string
	for _, device := range opts.devices {
		// Get the device specification
		d := registry.GetDevice(device)
		if d == nil {
			unresolved = append(unresolved, device)
			continue
		}

		if _, ok := specs[d.GetSpec()]; !ok {
			specs[d.GetSpec()] = struct{}{}
			specEdits := &cdi.ContainerEdits{
				ContainerEdits: &d.GetSpec().ContainerEdits,
			}
			edits.Append(specEdits)
		}
		deviceEdits := &cdi.ContainerEdits{
			ContainerEdits: &d.ContainerEdits,
		}
		edits.Append(deviceEdits)
	}

	for _, entry := range edits.toFstab() {
		fmt.Fprintln(outputWriter, entry.String())
	}

	return nil
}

// applyOCIMode reads an OCI spec, applies CDI device modifications, and writes the modified spec.
func (m command) applyOCIMode(ctx context.Context, opts *options, registry *cdi.Cache, outputWriter io.Writer) error {
	// Read the input OCI spec
	ociSpec, err := m.readOCISpec(opts.input)
	if err != nil {
		return fmt.Errorf("failed to read OCI spec: %v", err)
	}

	// Apply CDI device modifications to the OCI spec
	m.logger.Infof("Applying CDI devices to OCI spec: %v", opts.devices)
	unresolved, err := registry.InjectDevices(ociSpec, opts.devices...)
	if err != nil {
		return fmt.Errorf("failed to inject CDI devices: %v", err)
	}
	if len(unresolved) > 0 {
		m.logger.Warningf("Unresolved devices: %v", unresolved)
	}

	// Write the modified OCI spec
	if err := m.writeOCISpec(ociSpec, outputWriter); err != nil {
		return fmt.Errorf("failed to write OCI spec: %v", err)
	}

	return nil
}

// readOCISpec reads an OCI runtime spec from a file or stdin.
func (m command) readOCISpec(input string) (*specs.Spec, error) {
	reader, err := m.getInputReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	var spec specs.Spec
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode OCI spec: %v", err)
	}

	return &spec, nil
}

// writeOCISpec writes an OCI runtime spec to the output writer.
func (m command) writeOCISpec(spec *specs.Spec, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("failed to encode OCI spec: %v", err)
	}
	return nil
}

// getOutputWriter returns an io.Writer based on the output path.
// If output is empty or "-", it returns os.Stdout.
// Otherwise, it creates and returns a file writer.
func (m command) getOutputWriter(output string) (io.WriteCloser, error) {
	if output == "" || output == "-" {
		return writerWithCloser(os.Stdout), nil
	}

	return os.Create(output)
}

// getInputReader returns an io.Reader based on the output path.
// If output is empty or "-", it returns os.Stdin.
// Otherwise, it creates and returns a file writer.
func (m command) getInputReader(input string) (io.ReadCloser, error) {
	if input == "" || input == "-" {
		return readerWithCloser(os.Stdin), nil
	}

	return os.Open(input)
}

func readerWithCloser(r io.Reader) io.ReadCloser {
	if closer, ok := r.(io.ReadCloser); ok {
		return closer
	}
	return &noopCloser{Reader: r}
}

func writerWithCloser(w io.Writer) io.WriteCloser {
	if closer, ok := w.(io.WriteCloser); ok {
		return closer
	}
	return &noopCloser{Writer: w}
}

type noopCloser struct {
	io.Writer
	io.Reader
}

func (c *noopCloser) Close() error {
	return nil
}
