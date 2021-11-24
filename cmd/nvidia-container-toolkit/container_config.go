package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

var envSwarmGPU *string

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
	Requirements       []string
	DisableRequire     bool
}

type containerConfig struct {
	Pid    int
	Rootfs string
	Env    map[string]string
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

func parseCudaVersion(cudaVersion string) (vmaj, vmin, vpatch uint32) {
	if _, err := fmt.Sscanf(cudaVersion, "%d.%d.%d\n", &vmaj, &vmin, &vpatch); err != nil {
		vpatch = 0
		if _, err := fmt.Sscanf(cudaVersion, "%d.%d\n", &vmaj, &vmin); err != nil {
			vmin = 0
			if _, err := fmt.Sscanf(cudaVersion, "%d\n", &vmaj); err != nil {
				log.Panicln("invalid CUDA version:", cudaVersion)
			}
		}
	}

	return
}

func getEnvMap(e []string) (m map[string]string) {
	m = make(map[string]string)
	for _, s := range e {
		p := strings.SplitN(s, "=", 2)
		if len(p) != 2 {
			log.Panicln("environment error")
		}
		m[p[0]] = p[1]
	}
	return
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
	// If v1.1.0-rc1 <= OCI version < v1.0.0-rc5 parse s.Process.Capabilities as:
	// github.com/opencontainers/runtime-spec/blob/v1.0.0-rc1/specs-go/config.go#L30-L54
	rc1cmp := semver.Compare("v"+*s.Version, "v1.0.0-rc1")
	rc5cmp := semver.Compare("v"+*s.Version, "v1.0.0-rc5")
	if (rc1cmp == 1 || rc1cmp == 0) && (rc5cmp == -1) {
		err := json.Unmarshal(*s.Process.Capabilities, &caps)
		if err != nil {
			log.Panicln("could not decode Process.Capabilities in OCI spec:", err)
		}
		// Otherwise, parse s.Process.Capabilities as:
		// github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L30-L54
	} else {
		var lc LinuxCapabilities
		err := json.Unmarshal(*s.Process.Capabilities, &lc)
		if err != nil {
			log.Panicln("could not decode Process.Capabilities in OCI spec:", err)
		}
		// We only make sure that the bounding capabibility set has
		// CAP_SYS_ADMIN. This allows us to make sure that the container was
		// actually started as '--privileged', but also allow non-root users to
		// access the privileged NVIDIA capabilities.
		caps = lc.Bounding
	}

	for _, c := range caps {
		if c == capSysAdmin {
			return true
		}
	}

	return false
}

func isLegacyCUDAImage(env map[string]string) bool {
	legacyCudaVersion := env[envCUDAVersion]
	cudaRequire := env[envNVRequireCUDA]
	return len(legacyCudaVersion) > 0 && len(cudaRequire) == 0
}

func getDevicesFromEnvvar(env map[string]string, legacyImage bool) *string {
	// Build a list of envvars to consider.
	envVars := []string{envNVVisibleDevices}
	if envSwarmGPU != nil {
		// The Swarm envvar has higher precedence.
		envVars = append([]string{*envSwarmGPU}, envVars...)
	}

	// Grab a reference to devices from the first envvar
	// in the list that actually exists in the environment.
	var devices *string
	for _, envVar := range envVars {
		if devs, ok := env[envVar]; ok {
			devices = &devs
			break
		}
	}

	// Environment variable unset with legacy image: default to "all".
	if devices == nil && legacyImage {
		all := "all"
		return &all
	}

	// Environment variable unset or empty or "void": return nil
	if devices == nil || len(*devices) == 0 || *devices == "void" {
		return nil
	}

	// Environment variable set to "none": reset to "".
	if *devices == "none" {
		empty := ""
		return &empty
	}

	// Any other value.
	return devices
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

func getDevices(hookConfig *HookConfig, env map[string]string, mounts []Mount, privileged bool, legacyImage bool) *string {
	// If enabled, try and get the device list from volume mounts first
	if hookConfig.AcceptDeviceListAsVolumeMounts {
		devices := getDevicesFromMounts(mounts)
		if devices != nil {
			return devices
		}
	}

	// Fallback to reading from the environment variable if privileges are correct
	devices := getDevicesFromEnvvar(env, legacyImage)
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

func getMigConfigDevices(env map[string]string) *string {
	if devices, ok := env[envNVMigConfigDevices]; ok {
		return &devices
	}
	return nil
}

func getMigMonitorDevices(env map[string]string) *string {
	if devices, ok := env[envNVMigMonitorDevices]; ok {
		return &devices
	}
	return nil
}

func getDriverCapabilities(env map[string]string, supportedDriverCapabilities DriverCapabilities, legacyImage bool) DriverCapabilities {
	// We use the default driver capabilities by default. This is filtered to only include the
	// supported capabilities
	capabilities := supportedDriverCapabilities.Intersection(defaultDriverCapabilities)

	capsEnv, capsEnvSpecified := env[envNVDriverCapabilities]

	if !capsEnvSpecified && legacyImage {
		// Environment variable unset with legacy image: set all capabilities.
		return supportedDriverCapabilities
	}

	if capsEnvSpecified && len(capsEnv) > 0 {
		// If the envvironment variable is specified and is non-empty, use the capabilities value
		envCapabilities := DriverCapabilities(capsEnv)
		capabilities = supportedDriverCapabilities.Intersection(envCapabilities)
		if envCapabilities != all && capabilities != envCapabilities {
			log.Panicln(fmt.Errorf("unsupported capabilities found in '%v' (allowed '%v')", envCapabilities, capabilities))
		}
	}

	return capabilities
}

func getRequirements(env map[string]string, legacyImage bool) []string {
	// All variables with the "NVIDIA_REQUIRE_" prefix are passed to nvidia-container-cli
	var requirements []string
	for name, value := range env {
		if strings.HasPrefix(name, envNVRequirePrefix) {
			requirements = append(requirements, value)
		}
	}
	if legacyImage {
		vmaj, vmin, _ := parseCudaVersion(env[envCUDAVersion])
		cudaRequire := fmt.Sprintf("cuda>=%d.%d", vmaj, vmin)
		requirements = append(requirements, cudaRequire)
	}
	return requirements
}

func getNvidiaConfig(hookConfig *HookConfig, env map[string]string, mounts []Mount, privileged bool) *nvidiaConfig {
	legacyImage := isLegacyCUDAImage(env)

	var devices string
	if d := getDevices(hookConfig, env, mounts, privileged, legacyImage); d != nil {
		devices = *d
	} else {
		// 'nil' devices means this is not a GPU container.
		return nil
	}

	var migConfigDevices string
	if d := getMigConfigDevices(env); d != nil {
		migConfigDevices = *d
	}
	if !privileged && migConfigDevices != "" {
		log.Panicln("cannot set MIG_CONFIG_DEVICES in non privileged container")
	}

	var migMonitorDevices string
	if d := getMigMonitorDevices(env); d != nil {
		migMonitorDevices = *d
	}
	if !privileged && migMonitorDevices != "" {
		log.Panicln("cannot set MIG_MONITOR_DEVICES in non privileged container")
	}

	driverCapabilities := getDriverCapabilities(env, hookConfig.SupportedDriverCapabilities, legacyImage).String()

	requirements := getRequirements(env, legacyImage)

	// Don't fail on invalid values.
	disableRequire, _ := strconv.ParseBool(env[envNVDisableRequire])

	return &nvidiaConfig{
		Devices:            devices,
		MigConfigDevices:   migConfigDevices,
		MigMonitorDevices:  migMonitorDevices,
		DriverCapabilities: driverCapabilities,
		Requirements:       requirements,
		DisableRequire:     disableRequire,
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

	env := getEnvMap(s.Process.Env)
	privileged := isPrivileged(s)
	envSwarmGPU = hook.SwarmResource
	return containerConfig{
		Pid:    h.Pid,
		Rootfs: s.Root.Path,
		Env:    env,
		Nvidia: getNvidiaConfig(&hook, env, s.Mounts, privileged),
	}
}
