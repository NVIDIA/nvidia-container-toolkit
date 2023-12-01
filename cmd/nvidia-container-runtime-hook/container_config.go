package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/mod/semver"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

const (
	envCUDAVersion          = "CUDA_VERSION"
	envNVRequirePrefix      = "NVIDIA_REQUIRE_"
	envNVRequireCUDA        = envNVRequirePrefix + "CUDA"
	envNVDisableRequire     = "NVIDIA_DISABLE_REQUIRE"
	envNVVisibleDevices     = "NVIDIA_VISIBLE_DEVICES"
	envNVMigConfigDevices   = "NVIDIA_MIG_CONFIG_DEVICES"
	envNVMigMonitorDevices  = "NVIDIA_MIG_MONITOR_DEVICES"
	envNVDriverCapabilities = "NVIDIA_DRIVER_CAPABILITIES"
)

const (
	capSysAdmin = "CAP_SYS_ADMIN"
)

const (
	deviceListAsVolumeMountsRoot = "/var/run/nvidia-container-devices"
)

type nvidiaConfig struct {
	Devices            string
	MigConfigDevices   string
	MigMonitorDevices  string
	DriverCapabilities string
	// Requirements defines the requirements DSL for the container to run.
	// This is empty if no specific requirements are needed, or if requirements are
	// explicitly disabled.
	Requirements []string
}

type containerConfig struct {
	Pid    int
	Rootfs string
	Image  image.CUDA
	Nvidia *nvidiaConfig
}

// Root from OCI runtime spec
// github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L94-L100
type Root struct {
	Path string `json:"path"`
}

// Process from OCI runtime spec
// github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L30-L57
type Process struct {
	Env          []string         `json:"env,omitempty"`
	Capabilities *json.RawMessage `json:"capabilities,omitempty" platform:"linux"`
}

// LinuxCapabilities from OCI runtime spec
// https://github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L61
type LinuxCapabilities struct {
	Bounding    []string `json:"bounding,omitempty" platform:"linux"`
	Effective   []string `json:"effective,omitempty" platform:"linux"`
	Inheritable []string `json:"inheritable,omitempty" platform:"linux"`
	Permitted   []string `json:"permitted,omitempty" platform:"linux"`
	Ambient     []string `json:"ambient,omitempty" platform:"linux"`
}

// Mount from OCI runtime spec
// https://github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L103
type Mount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type,omitempty" platform:"linux,solaris"`
	Source      string   `json:"source,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// Spec from OCI runtime spec
// We use pointers to structs, similarly to the latest version of runtime-spec:
// https://github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L5-L28
type Spec struct {
	Version *string  `json:"ociVersion"`
	Process *Process `json:"process,omitempty"`
	Root    *Root    `json:"root,omitempty"`
	Mounts  []Mount  `json:"mounts,omitempty"`
}

// HookState holds state information about the hook
type HookState struct {
	Pid int `json:"pid,omitempty"`
	// After 17.06, runc is using the runtime spec:
	// github.com/docker/runc/blob/17.06/libcontainer/configs/config.go#L262-L263
	// github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/state.go#L3-L17
	Bundle string `json:"bundle"`
	// Before 17.06, runc used a custom struct that didn't conform to the spec:
	// github.com/docker/runc/blob/17.03.x/libcontainer/configs/config.go#L245-L252
	BundlePath string `json:"bundlePath"`
}

func loadSpec(path string) (spec *Spec) {
	f, err := os.Open(path)
	if err != nil {
		log.Panicln("could not open OCI spec:", err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&spec); err != nil {
		log.Panicln("could not decode OCI spec:", err)
	}
	if spec.Version == nil {
		log.Panicln("Version is empty in OCI spec")
	}
	if spec.Process == nil {
		log.Panicln("Process is empty in OCI spec")
	}
	if spec.Root == nil {
		log.Panicln("Root is empty in OCI spec")
	}
	return
}

func isPrivileged(s *Spec) bool {
	if s.Process.Capabilities == nil {
		return false
	}

	var caps []string
	// If v1.0.0-rc1 <= OCI version < v1.0.0-rc5 parse s.Process.Capabilities as:
	// github.com/opencontainers/runtime-spec/blob/v1.0.0-rc1/specs-go/config.go#L30-L54
	rc1cmp := semver.Compare("v"+*s.Version, "v1.0.0-rc1")
	rc5cmp := semver.Compare("v"+*s.Version, "v1.0.0-rc5")
	if (rc1cmp == 1 || rc1cmp == 0) && (rc5cmp == -1) {
		err := json.Unmarshal(*s.Process.Capabilities, &caps)
		if err != nil {
			log.Panicln("could not decode Process.Capabilities in OCI spec:", err)
		}
		for _, c := range caps {
			if c == capSysAdmin {
				return true
			}
		}
		return false
	}

	// Otherwise, parse s.Process.Capabilities as:
	// github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L30-L54
	process := specs.Process{
		Env: s.Process.Env,
	}

	err := json.Unmarshal(*s.Process.Capabilities, &process.Capabilities)
	if err != nil {
		log.Panicln("could not decode Process.Capabilities in OCI spec:", err)
	}

	fullSpec := specs.Spec{
		Version: *s.Version,
		Process: &process,
	}

	return image.IsPrivileged(&fullSpec)
}

func getDevicesFromEnvvar(image image.CUDA, swarmResourceEnvvars []string) *string {
	// We check if the image has at least one of the Swarm resource envvars defined and use this
	// if specified.
	var hasSwarmEnvvar bool
	for _, envvar := range swarmResourceEnvvars {
		if image.HasEnvvar(envvar) {
			hasSwarmEnvvar = true
			break
		}
	}

	var devices []string
	if hasSwarmEnvvar {
		devices = image.DevicesFromEnvvars(swarmResourceEnvvars...).List()
	} else {
		devices = image.DevicesFromEnvvars(envNVVisibleDevices).List()
	}

	if len(devices) == 0 {
		return nil
	}

	devicesString := strings.Join(devices, ",")

	return &devicesString
}

func getDevicesFromMounts(mounts []Mount) *string {
	var devices []string
	for _, m := range mounts {
		root := filepath.Clean(deviceListAsVolumeMountsRoot)
		source := filepath.Clean(m.Source)
		destination := filepath.Clean(m.Destination)

		// Only consider mounts who's host volume is /dev/null
		if source != "/dev/null" {
			continue
		}
		// Only consider container mount points that begin with 'root'
		if len(destination) < len(root) {
			continue
		}
		if destination[:len(root)] != root {
			continue
		}
		// Grab the full path beyond 'root' and add it to the list of devices
		device := destination[len(root):]
		if len(device) > 0 && device[0] == '/' {
			device = device[1:]
		}
		if len(device) == 0 {
			continue
		}
		devices = append(devices, device)
	}

	if devices == nil {
		return nil
	}

	ret := strings.Join(devices, ",")
	return &ret
}

func getDevices(hookConfig *HookConfig, image image.CUDA, mounts []Mount, privileged bool) *string {
	// If enabled, try and get the device list from volume mounts first
	if hookConfig.AcceptDeviceListAsVolumeMounts {
		devices := getDevicesFromMounts(mounts)
		if devices != nil {
			return devices
		}
	}

	// Fallback to reading from the environment variable if privileges are correct
	devices := getDevicesFromEnvvar(image, hookConfig.getSwarmResourceEnvvars())
	if devices == nil {
		return nil
	}
	if privileged || hookConfig.AcceptEnvvarUnprivileged {
		return devices
	}

	configName := hookConfig.getConfigOption("AcceptEnvvarUnprivileged")
	log.Printf("Ignoring devices specified in NVIDIA_VISIBLE_DEVICES (privileged=%v, %v=%v) ", privileged, configName, hookConfig.AcceptEnvvarUnprivileged)

	return nil
}

func getMigConfigDevices(image image.CUDA) *string {
	return getMigDevices(image, envNVMigConfigDevices)
}

func getMigMonitorDevices(image image.CUDA) *string {
	return getMigDevices(image, envNVMigMonitorDevices)
}

func getMigDevices(image image.CUDA, envvar string) *string {
	if !image.HasEnvvar(envvar) {
		return nil
	}
	devices := image.Getenv(envvar)
	return &devices
}

func (c *HookConfig) getDriverCapabilities(cudaImage image.CUDA, legacyImage bool) image.DriverCapabilities {
	// We use the default driver capabilities by default. This is filtered to only include the
	// supported capabilities
	supportedDriverCapabilities := image.NewDriverCapabilities(c.SupportedDriverCapabilities)

	capabilities := supportedDriverCapabilities.Intersection(image.DefaultDriverCapabilities)

	capsEnvSpecified := cudaImage.HasEnvvar(envNVDriverCapabilities)
	capsEnv := cudaImage.Getenv(envNVDriverCapabilities)

	if !capsEnvSpecified && legacyImage {
		// Environment variable unset with legacy image: set all capabilities.
		return supportedDriverCapabilities
	}

	if capsEnvSpecified && len(capsEnv) > 0 {
		// If the envvironment variable is specified and is non-empty, use the capabilities value
		envCapabilities := image.NewDriverCapabilities(capsEnv)
		capabilities = supportedDriverCapabilities.Intersection(envCapabilities)
		if !envCapabilities.IsAll() && len(capabilities) != len(envCapabilities) {
			log.Panicln(fmt.Errorf("unsupported capabilities found in '%v' (allowed '%v')", envCapabilities, capabilities))
		}
	}

	return capabilities
}

func getNvidiaConfig(hookConfig *HookConfig, image image.CUDA, mounts []Mount, privileged bool) *nvidiaConfig {
	legacyImage := image.IsLegacy()

	var devices string
	if d := getDevices(hookConfig, image, mounts, privileged); d != nil {
		devices = *d
	} else {
		// 'nil' devices means this is not a GPU container.
		return nil
	}

	var migConfigDevices string
	if d := getMigConfigDevices(image); d != nil {
		migConfigDevices = *d
	}
	if !privileged && migConfigDevices != "" {
		log.Panicln("cannot set MIG_CONFIG_DEVICES in non privileged container")
	}

	var migMonitorDevices string
	if d := getMigMonitorDevices(image); d != nil {
		migMonitorDevices = *d
	}
	if !privileged && migMonitorDevices != "" {
		log.Panicln("cannot set MIG_MONITOR_DEVICES in non privileged container")
	}

	driverCapabilities := hookConfig.getDriverCapabilities(image, legacyImage).String()

	requirements, err := image.GetRequirements()
	if err != nil {
		log.Panicln("failed to get requirements", err)
	}

	return &nvidiaConfig{
		Devices:            devices,
		MigConfigDevices:   migConfigDevices,
		MigMonitorDevices:  migMonitorDevices,
		DriverCapabilities: driverCapabilities,
		Requirements:       requirements,
	}
}

func getContainerConfig(hook HookConfig) (config containerConfig) {
	var h HookState
	d := json.NewDecoder(os.Stdin)
	if err := d.Decode(&h); err != nil {
		log.Panicln("could not decode container state:", err)
	}

	b := h.Bundle
	if len(b) == 0 {
		b = h.BundlePath
	}

	s := loadSpec(path.Join(b, "config.json"))

	image, err := image.New(
		image.WithEnv(s.Process.Env),
		image.WithDisableRequire(hook.DisableRequire),
	)
	if err != nil {
		log.Panicln(err)
	}

	privileged := isPrivileged(s)
	return containerConfig{
		Pid:    h.Pid,
		Rootfs: s.Root.Path,
		Image:  image,
		Nvidia: getNvidiaConfig(&hook, image, s.Mounts, privileged),
	}
}
