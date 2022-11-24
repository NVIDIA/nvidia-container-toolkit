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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
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

	formatJSON = "json"
	formatYAML = "yaml"
)

type command struct {
	logger *logrus.Logger
}

type config struct {
	output   string
	format   string
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

	return nil
}

func (m command) run(c *cli.Context, cfg *config) error {
	spec, err := m.generateSpec()
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

	err = writeToOutput(cfg.jsonMode, data, outputTo)
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

func writeToOutput(jsonMode bool, data []byte, output io.Writer) error {
	if !jsonMode {
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

func (m command) generateSpec() (*specs.Spec, error) {
	nvmllib := nvml.New()
	if r := nvmllib.Init(); r != nvml.SUCCESS {
		return nil, r
	}
	defer nvmllib.Shutdown()

	devicelib := device.New(device.WithNvml(nvmllib))

	spec := specs.Spec{
		Version:        "0.4.0",
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

		graphicsEdits, err := m.editsForGraphicsDevice(d)
		if err != nil {
			return fmt.Errorf("failed to generate CDI spec for DRM devices associated with device %v: %v", i, err)
		}

		// We add the device nodes and hooks edits for the DRM devices; Mounts are added globally
		for _, dn := range graphicsEdits.DeviceNodes {
			device.ContainerEdits.DeviceNodes = append(device.ContainerEdits.DeviceNodes, dn)
		}
		for _, h := range graphicsEdits.Hooks {
			device.ContainerEdits.Hooks = append(device.ContainerEdits.Hooks, h)
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

	// We create an "all" device with all the discovered device nodes
	var allDeviceNodes []*specs.DeviceNode
	for _, d := range spec.Devices {
		for _, dn := range d.ContainerEdits.DeviceNodes {
			allDeviceNodes = append(allDeviceNodes, dn)
		}
	}
	all := specs.Device{
		Name: "all",
		ContainerEdits: specs.ContainerEdits{
			DeviceNodes: allDeviceNodes,
		},
	}

	spec.Devices = append(spec.Devices, all)
	spec.ContainerEdits.DeviceNodes = m.getExistingMetaDeviceNodes()

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

	graphicsEdits, err := m.editsForGraphicsDevice(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate edits for graphics libraries: %v", err)
	}

	libOptions := []string{
		"ro",
		"nosuid",
		"nodev",
		"bind",
	}
	ipcOptions := append(libOptions, "noexec")

	spec.ContainerEdits.Mounts = append(
		generateMountsForPaths(libOptions, libraries, binaries),
		generateMountsForPaths(ipcOptions, ipcs)...,
	)

	spec.ContainerEdits.Mounts = append(spec.ContainerEdits.Mounts, graphicsEdits.Mounts...)

	ldcacheUpdateHook := m.generateUpdateLdCacheHook(libraries)

	deviceFolderPermissionHooks, err := m.generateDeviceFolderPermissionHooks(ldcacheUpdateHook.Path, allDeviceNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to generated permission hooks for device nodes: %v", err)
	}

	spec.ContainerEdits.Hooks = append([]*specs.Hook{ldcacheUpdateHook}, deviceFolderPermissionHooks...)

	return &spec, nil
}

func generateEditsForDevice(name string, d deviceInfo) (specs.Device, error) {
	deviceNodePaths, err := d.GetDeviceNodes()
	if err != nil {
		return specs.Device{}, fmt.Errorf("failed to get paths for device: %v", err)
	}

	deviceNodes := getDeviceNodesFromPaths(deviceNodePaths)

	device := specs.Device{
		Name: name,
		ContainerEdits: specs.ContainerEdits{
			DeviceNodes: deviceNodes,
		},
	}

	return device, nil
}

func (m command) editsForGraphicsDevice(device device.Device) (*specs.ContainerEdits, error) {
	selectedDevice := image.NewVisibleDevices("none")
	if device != nil {
		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("error getting device UUID: %v", ret)
		}
		selectedDevice = image.NewVisibleDevices(uuid)
	}
	cfg := discover.Config{
		Root:                                    "",
		NVIDIAContainerToolkitCLIExecutablePath: "nvidia-ctk",
	}
	// Create a discoverer for the single device:
	d, err := discover.NewGraphicsDiscoverer(m.logger, selectedDevice, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error constructing discoverer: %v", err)
	}

	devices, err := d.Devices()
	if err != nil {
		return nil, fmt.Errorf("error getting DRM devices: %v", err)
	}

	var deviceNodes []*specs.DeviceNode
	for _, d := range devices {
		dn := specs.DeviceNode{
			Path:     d.Path,
			HostPath: d.HostPath,
		}
		deviceNodes = append(deviceNodes, &dn)
	}

	hooks, err := d.Hooks()
	if err != nil {
		return nil, fmt.Errorf("error getting hooks: %v", err)
	}

	var cdiHooks []*specs.Hook
	for _, h := range hooks {
		cdiHook := specs.Hook{
			HookName: h.Lifecycle,
			Path:     h.Path,
			Args:     h.Args,
		}
		cdiHooks = append(cdiHooks, &cdiHook)
	}

	mounts, err := d.Mounts()
	if err != nil {
		return nil, fmt.Errorf("error getting mounts: %v", err)
	}

	var cdiMounts []*specs.Mount
	for _, m := range mounts {
		cdiMount := specs.Mount{
			ContainerPath: m.Path,
			HostPath:      m.HostPath,
			Options: []string{
				"ro",
				"nosuid",
				"nodev",
				"bind",
			},
			Type: "bind",
		}
		cdiMounts = append(cdiMounts, &cdiMount)
	}

	edits := specs.ContainerEdits{
		DeviceNodes: deviceNodes,
		Hooks:       cdiHooks,
		Mounts:      cdiMounts,
	}

	return &edits, nil
}

func (m command) getExistingMetaDeviceNodes() []*specs.DeviceNode {
	metaDeviceNodePaths := []string{
		"/dev/nvidia-modeset",
		"/dev/nvidia-uvm-tools",
		"/dev/nvidia-uvm",
		"/dev/nvidiactl",
	}

	var existingDeviceNodePaths []string
	for _, p := range metaDeviceNodePaths {
		if _, err := os.Stat(p); err != nil {
			m.logger.Infof("Ignoring missing meta device %v", p)
			continue
		}
		existingDeviceNodePaths = append(existingDeviceNodePaths, p)
	}

	return getDeviceNodesFromPaths(existingDeviceNodePaths)
}

func getDeviceNodesFromPaths(deviceNodePaths []string) []*specs.DeviceNode {
	var deviceNodes []*specs.DeviceNode
	for _, p := range deviceNodePaths {
		deviceNode := specs.DeviceNode{
			Path: p,
		}
		deviceNodes = append(deviceNodes, &deviceNode)
	}

	return deviceNodes
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

func generateMountsForPaths(options []string, pathSets ...[]string) []*specs.Mount {
	var mounts []*specs.Mount
	for _, paths := range pathSets {
		for _, p := range paths {
			mount := specs.Mount{
				HostPath: p,
				// We may want to adjust the container path
				ContainerPath: p,
				Type:          "bind",
				Options:       options,
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

func (m command) generateDeviceFolderPermissionHooks(nvidiaCTKPath string, deviceNodes []*specs.DeviceNode) ([]*specs.Hook, error) {
	var deviceFolders []string
	seen := make(map[string]bool)

	for _, dn := range deviceNodes {
		if !strings.HasPrefix(dn.Path, "/dev") {
			m.logger.Warningf("Skipping unexpected device folder path for device %v", dn.Path)
			continue
		}
		for df := filepath.Dir(dn.Path); df != "/dev"; df = filepath.Dir(df) {
			if seen[df] {
				continue
			}
			deviceFolders = append(deviceFolders, df)
			seen[df] = true
		}
	}

	foldersByMode := make(map[string][]string)
	for _, p := range deviceFolders {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("failed to get info for path %v: %v", p, err)
		}
		mode := fmt.Sprintf("%o", info.Mode().Perm())
		foldersByMode[mode] = append(foldersByMode[mode], p)
	}

	var hooks []*specs.Hook
	for mode, folders := range foldersByMode {
		args := []string{filepath.Base(nvidiaCTKPath), "hook", "chmod", "--mode", mode}
		for _, folder := range folders {
			args = append(args, "--path", folder)
		}
		hook := specs.Hook{
			HookName: cdi.CreateContainerHook,
			Path:     nvidiaCTKPath,
			Args:     args,
		}

		hooks = append(hooks, &hook)
	}

	return hooks, nil
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
