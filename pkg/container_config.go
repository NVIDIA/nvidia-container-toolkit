package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
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
	allDriverCapabilities     = "compute,compat32,graphics,utility,video,display,ngx"
	defaultDriverCapabilities = "utility"
)

const (
	capSysAdmin = "CAP_SYS_ADMIN"
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

// Spec from OCI runtime spec
// We use pointers to structs, similarly to the latest version of runtime-spec:
// https://github.com/opencontainers/runtime-spec/blob/v1.0.0/specs-go/config.go#L5-L28
type Spec struct {
	Version *string  `json:"ociVersion"`
	Process *Process `json:"process,omitempty"`
	Root    *Root    `json:"root,omitempty"`
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

func getEnvMap(e []string, config CLIConfig) (m map[string]string) {
	m = make(map[string]string)
	for _, s := range e {
		p := strings.SplitN(s, "=", 2)
		if len(p) != 2 {
			log.Panicln("environment error")
		}
		m[p[0]] = p[1]
	}
	if config.AlphaMergeVisibleDevicesEnvvars {
		var mergable []string
		for k, v := range m {
			if strings.HasPrefix(k, envNVVisibleDevices+"_") {
				mergable = append(mergable, v)
			}
		}
		if len(mergable) > 0 {
			m[envNVVisibleDevices] = strings.Join(mergable, ",")
		}
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

func getDevices(env map[string]string) *string {
	gpuVars := []string{envNVVisibleDevices}
	if envSwarmGPU != nil {
		// The Swarm resource has higher precedence.
		gpuVars = append([]string{*envSwarmGPU}, gpuVars...)
	}

	for _, gpuVar := range gpuVars {
		if devices, ok := env[gpuVar]; ok {
			return &devices
		}
	}
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

func getDriverCapabilities(env map[string]string) *string {
	if capabilities, ok := env[envNVDriverCapabilities]; ok {
		return &capabilities
	}
	return nil
}

func getRequirements(env map[string]string) []string {
	// All variables with the "NVIDIA_REQUIRE_" prefix are passed to nvidia-container-cli
	var requirements []string
	for name, value := range env {
		if strings.HasPrefix(name, envNVRequirePrefix) {
			requirements = append(requirements, value)
		}
	}
	return requirements
}

// Mimic the new CUDA images if no capabilities or devices are specified.
func getNvidiaConfigLegacy(env map[string]string, privileged bool) *nvidiaConfig {
	var devices string
	if d := getDevices(env); d == nil {
		// Environment variable unset: default to "all".
		devices = "all"
	} else if len(*d) == 0 || *d == "void" {
		// Environment variable empty or "void": not a GPU container.
		return nil
	} else {
		// Environment variable non-empty and not "void".
		devices = *d
	}
	if devices == "none" {
		devices = ""
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

	var driverCapabilities string
	if c := getDriverCapabilities(env); c == nil {
		// Environment variable unset: default to "all".
		driverCapabilities = allDriverCapabilities
	} else if len(*c) == 0 {
		// Environment variable empty: use default capability.
		driverCapabilities = defaultDriverCapabilities
	} else {
		// Environment variable non-empty.
		driverCapabilities = *c
	}
	if driverCapabilities == "all" {
		driverCapabilities = allDriverCapabilities
	}

	requirements := getRequirements(env)

	vmaj, vmin, _ := parseCudaVersion(env[envCUDAVersion])
	cudaRequire := fmt.Sprintf("cuda>=%d.%d", vmaj, vmin)
	requirements = append(requirements, cudaRequire)

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

func getNvidiaConfig(env map[string]string, privileged bool) *nvidiaConfig {
	if isLegacyCUDAImage(env) {
		return getNvidiaConfigLegacy(env, privileged)
	}

	var devices string
	d := getDevices(env)
	if d == nil || len(*d) == 0 || *d == "void" {
		// Environment variable unset or empty or "void": not a GPU container.
		return nil
	}

	// Environment variable non-empty and not "void".
	devices = *d

	if devices == "none" {
		devices = ""
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

	var driverCapabilities string
	if c := getDriverCapabilities(env); c == nil || len(*c) == 0 {
		// Environment variable unset or set but empty: use default capability.
		driverCapabilities = defaultDriverCapabilities
	} else {
		// Environment variable set and non-empty.
		driverCapabilities = *c
	}
	if driverCapabilities == "all" {
		driverCapabilities = allDriverCapabilities
	}

	requirements := getRequirements(env)

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

	env := getEnvMap(s.Process.Env, hook.NvidiaContainerCLI)
	privileged := isPrivileged(s)
	envSwarmGPU = hook.SwarmResource
	return containerConfig{
		Pid:    h.Pid,
		Rootfs: s.Root.Path,
		Env:    env,
		Nvidia: getNvidiaConfig(env, privileged),
	}
}
