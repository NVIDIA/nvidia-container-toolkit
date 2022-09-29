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

package cdi

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	specs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
	"sigs.k8s.io/yaml"
)

const (
	nvidiaCTKExecutable      = "nvidia-ctk"
	nvidiaCTKDefaultFilePath = "/usr/bin/" + nvidiaCTKExecutable
)

type command struct {
	logger *logrus.Logger
}

type config struct {
	output   string
	jsonMode bool
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
		Name:  "generate-cdi",
		Usage: "Generate CDI specifications for use with CDI-enabled runtimes",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Usage:       "Specify the file to output the generated CDI specification to. If this is '-' or '' the specification is output to STDOUT",
			Destination: &cfg.output,
		},
		&cli.BoolFlag{
			Name:        "json",
			Usage:       "Output the generated CDI spec in JSON mode instead of YAML",
			Destination: &cfg.jsonMode,
		},
	}

	return &c
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.generateSpec()
	if err != nil {
		return fmt.Errorf("failed to generate CDI spec: %v", err)
	}

	var outputTo io.Writer
	if cfg.output == "" || cfg.output == "-" {
		outputTo = os.Stdout
	} else {
		outputFile, err := os.Create(cfg.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outputFile.Close()
		outputTo = outputFile
	}

	if filepath.Ext(cfg.output) == ".json" {
		cfg.jsonMode = true
	} else if filepath.Ext(cfg.output) == ".yaml" || filepath.Ext(cfg.output) == ".yml" {
		cfg.jsonMode = false
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal CDI spec: %v", err)
	}

	if cfg.jsonMode {
		data, err = yaml.YAMLToJSONStrict(data)
		if err != nil {
			return fmt.Errorf("failed to convert CDI spec from YAML to JSON: %v", err)
		}
	}

	_, err = outputTo.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	return nil
}

func (m command) generateSpec() (*specs.Spec, error) {
	nvmllib := nvml.New()
	if r := nvmllib.Init(); r != nvml.SUCCESS {
		return nil, r
	}
	defer nvmllib.Shutdown()

	devicelib := device.New(device.WithNvml(nvmllib))

	spec := specs.Spec{
		Version:        specs.CurrentVersion,
		Kind:           "nvidia.com/gpu",
		ContainerEdits: specs.ContainerEdits{},
	}
	err := devicelib.VisitDevices(func(i int, d device.Device) error {
		isMig, err := d.IsMigEnabled()
		if err != nil {
			return fmt.Errorf("failed to check whether device is MIG device: %v", err)
		}
		if isMig {
			return nil
		}
		device, err := generateEditsForDevice(newGPUDevice(i, d))
		if err != nil {
			return fmt.Errorf("failed to generate CDI spec for device %v: %v", i, err)
		}

		spec.Devices = append(spec.Devices, device)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CDI spec for GPU devices: %v", err)
	}

	err = devicelib.VisitMigDevices(func(i int, d device.Device, j int, m device.MigDevice) error {
		device, err := generateEditsForDevice(newMigDevice(i, j, m))
		if err != nil {
			return fmt.Errorf("failed to generate CDI spec for device %v: %v", i, err)
		}

		spec.Devices = append(spec.Devices, device)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("falied to generate CDI spec for MIG devices: %v", err)
	}

	libraries, err := m.findLibs(nvmllib)
	if err != nil {
		return nil, fmt.Errorf("failed to locate driver libraries: %v", err)
	}

	binaries, err := m.findBinaries()
	if err != nil {
		return nil, fmt.Errorf("failed to locate driver binaries: %v", err)
	}

	ipcs, err := m.findIPC()
	if err != nil {
		return nil, fmt.Errorf("failed to locate driver IPC sockets: %v", err)
	}

	spec.ContainerEdits.Mounts = generateMountsForPaths(libraries, binaries, ipcs)

	ldcacheUpdateHook := m.generateUpdateLdCacheHook(libraries)

	spec.ContainerEdits.Hooks = []*specs.Hook{ldcacheUpdateHook}

	return &spec, nil
}

func generateEditsForDevice(name string, d deviceInfo) (specs.Device, error) {
	var deviceNodes []*specs.DeviceNode

	deviceNodePaths, err := d.GetDeviceNodes()
	if err != nil {
		return specs.Device{}, fmt.Errorf("failed to get paths for device: %v", err)
	}
	for _, p := range deviceNodePaths {
		deviceNode := specs.DeviceNode{
			Path: p,
			// TODO: Set the host path dependent on the root
			HostPath: p,
		}
		deviceNodes = append(deviceNodes, &deviceNode)
	}
	device := specs.Device{
		Name: name,
		ContainerEdits: specs.ContainerEdits{
			DeviceNodes: deviceNodes,
		},
	}

	return device, nil
}

func (m command) findLibs(nvmllib nvml.Interface) ([]string, error) {
	version, r := nvmllib.SystemGetDriverVersion()
	if r != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to determine driver version: %v", r)
	}
	m.logger.Infof("Using driver version %v", version)

	cache, err := ldcache.New(m.logger, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load ldcache: %v", err)
	}

	libs32, libs64 := cache.List()

	var libs []string
	for _, l := range libs64 {
		if strings.HasSuffix(l, version) {
			m.logger.Infof("found 64-bit driver lib: %v", l)
			libs = append(libs, l)
		}
	}

	for _, l := range libs32 {
		if strings.HasSuffix(l, version) {
			m.logger.Infof("found 32-bit driver lib: %v", l)
			libs = append(libs, l)
		}
	}

	return libs, nil
}

func (m command) findBinaries() ([]string, error) {
	candidates := []string{
		"nvidia-smi",              /* System management interface */
		"nvidia-debugdump",        /* GPU coredump utility */
		"nvidia-persistenced",     /* Persistence mode utility */
		"nvidia-cuda-mps-control", /* Multi process service CLI */
		"nvidia-cuda-mps-server",  /* Multi process service server */
	}

	locator := lookup.NewExecutableLocator(m.logger, "")

	var binaries []string
	for _, c := range candidates {
		targets, err := locator.Locate(c)
		if err != nil {
			m.logger.Warningf("skipping %v: %v", c, err)
			continue
		}

		binaries = append(binaries, targets[0])
	}
	return binaries, nil
}

func (m command) findIPC() ([]string, error) {
	candidates := []string{
		"/var/run/nvidia-persistenced/socket",
		"/var/run/nvidia-fabricmanager/socket",
		// TODO: This can be controlled by the NV_MPS_PIPE_DIR envvar
		"/tmp/nvidia-mps",
	}

	locator := lookup.NewFileLocator(m.logger, "")

	var ipcs []string
	for _, c := range candidates {
		targets, err := locator.Locate(c)
		if err != nil {
			m.logger.Warningf("skipping %v: %v", c, err)
			continue
		}

		ipcs = append(ipcs, targets[0])
	}
	return ipcs, nil
}

func generateMountsForPaths(pathSets ...[]string) []*specs.Mount {
	var mounts []*specs.Mount
	for _, paths := range pathSets {
		for _, p := range paths {
			mount := specs.Mount{
				HostPath: p,
				// We may want to adjust the container path
				ContainerPath: p,
				Type:          "bind",
				Options: []string{
					"ro",
					"nosuid",
					"nodev",
					"bind",
				},
			}
			mounts = append(mounts, &mount)
		}
	}
	return mounts
}

func (m command) generateUpdateLdCacheHook(libraries []string) *specs.Hook {
	locator := lookup.NewExecutableLocator(m.logger, "")

	hook := discover.CreateLDCacheUpdateHook(
		m.logger,
		locator,
		nvidiaCTKExecutable,
		nvidiaCTKDefaultFilePath,
		libraries,
	)
	return &specs.Hook{
		HookName: hook.Lifecycle,
		Path:     hook.Path,
		Args:     hook.Args,
	}
}
