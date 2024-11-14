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

package image

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/mod/semver"
	"tags.cncf.io/container-device-interface/pkg/parser"
)

const (
	DeviceListAsVolumeMountsRoot = "/var/run/nvidia-container-devices"

	volumeMountDevicePrefixCDI  = "cdi/"
	volumeMountDevicePrefixImex = "imex/"
)

// CUDA represents a CUDA image that can be used for GPU computing. This wraps
// a map of environment variable to values that can be used to perform lookups
// such as requirements.
type CUDA struct {
	env    map[string]string
	mounts []specs.Mount
}

// NewCUDAImageFromSpec creates a CUDA image from the input OCI runtime spec.
// The process environment is read (if present) to construc the CUDA Image.
func NewCUDAImageFromSpec(spec *specs.Spec) (CUDA, error) {
	var env []string
	if spec != nil && spec.Process != nil {
		env = spec.Process.Env
	}

	return New(
		WithEnv(env),
		WithMounts(spec.Mounts),
	)
}

// NewCUDAImageFromEnv creates a CUDA image from the input environment. The environment
// is a list of strings of the form ENVAR=VALUE.
func NewCUDAImageFromEnv(env []string) (CUDA, error) {
	return New(WithEnv(env))
}

// Getenv returns the value of the specified environment variable.
// If the environment variable is not specified, an empty string is returned.
func (i CUDA) Getenv(key string) string {
	return i.env[key]
}

// HasEnvvar checks whether the specified envvar is defined in the image.
func (i CUDA) HasEnvvar(key string) bool {
	_, exists := i.env[key]
	return exists
}

// IsLegacy returns whether the associated CUDA image is a "legacy" image. An
// image is considered legacy if it has a CUDA_VERSION environment variable defined
// and no NVIDIA_REQUIRE_CUDA environment variable defined.
func (i CUDA) IsLegacy() bool {
	legacyCudaVersion := i.env[EnvVarCudaVersion]
	cudaRequire := i.env[EnvVarNvidiaRequireCuda]
	return len(legacyCudaVersion) > 0 && len(cudaRequire) == 0
}

// GetRequirements returns the requirements from all NVIDIA_REQUIRE_ environment
// variables.
func (i CUDA) GetRequirements() ([]string, error) {
	if i.HasDisableRequire() {
		return nil, nil
	}

	// All variables with the "NVIDIA_REQUIRE_" prefix are passed to nvidia-container-cli
	var requirements []string
	for name, value := range i.env {
		if strings.HasPrefix(name, NvidiaRequirePrefix) && !strings.HasPrefix(name, EnvVarNvidiaRequireJetpack) {
			requirements = append(requirements, value)
		}
	}
	if i.IsLegacy() {
		v, err := i.legacyVersion()
		if err != nil {
			return nil, fmt.Errorf("failed to get version: %v", err)
		}
		cudaRequire := fmt.Sprintf("cuda>=%s", v)
		requirements = append(requirements, cudaRequire)
	}
	return requirements, nil
}

// HasDisableRequire checks for the value of the NVIDIA_DISABLE_REQUIRE. If set
// to a valid (true) boolean value this can be used to disable the requirement checks
func (i CUDA) HasDisableRequire() bool {
	if disable, exists := i.env[EnvVarNvidiaDisableRequire]; exists {
		// i.logger.Debugf("NVIDIA_DISABLE_REQUIRE=%v; skipping requirement checks", disable)
		d, _ := strconv.ParseBool(disable)
		return d
	}

	return false
}

// DevicesFromEnvvars returns the devices requested by the image through environment variables
func (i CUDA) DevicesFromEnvvars(envVars ...string) VisibleDevices {
	// We concantenate all the devices from the specified env.
	var isSet bool
	var devices []string
	requested := make(map[string]bool)
	for _, envVar := range envVars {
		if devs, ok := i.env[envVar]; ok {
			isSet = true
			for _, d := range strings.Split(devs, ",") {
				trimmed := strings.TrimSpace(d)
				if len(trimmed) == 0 {
					continue
				}
				devices = append(devices, trimmed)
				requested[trimmed] = true
			}
		}
	}

	// Environment variable unset with legacy image: default to "all".
	if !isSet && len(devices) == 0 && i.IsLegacy() {
		return NewVisibleDevices("all")
	}

	// Environment variable unset or empty or "void": return nil
	if len(devices) == 0 || requested["void"] {
		return NewVisibleDevices("void")
	}

	return NewVisibleDevices(devices...)
}

// GetDriverCapabilities returns the requested driver capabilities.
func (i CUDA) GetDriverCapabilities() DriverCapabilities {
	env := i.env[EnvVarNvidiaDriverCapabilities]

	capabilities := make(DriverCapabilities)
	for _, c := range strings.Split(env, ",") {
		capabilities[DriverCapability(c)] = true
	}

	return capabilities
}

func (i CUDA) legacyVersion() (string, error) {
	cudaVersion := i.env[EnvVarCudaVersion]
	majorMinor, err := parseMajorMinorVersion(cudaVersion)
	if err != nil {
		return "", fmt.Errorf("invalid CUDA version %v: %v", cudaVersion, err)
	}

	return majorMinor, nil
}

func parseMajorMinorVersion(version string) (string, error) {
	vVersion := "v" + strings.TrimPrefix(version, "v")

	if !semver.IsValid(vVersion) {
		return "", fmt.Errorf("invalid version string")
	}

	majorMinor := strings.TrimPrefix(semver.MajorMinor(vVersion), "v")
	parts := strings.Split(majorMinor, ".")

	var err error
	_, err = strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid major version")
	}
	_, err = strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid minor version")
	}
	return majorMinor, nil
}

// OnlyFullyQualifiedCDIDevices returns true if all devices requested in the image are requested as CDI devices/
func (i CUDA) OnlyFullyQualifiedCDIDevices() bool {
	var hasCDIdevice bool
	for _, device := range i.VisibleDevicesFromEnvVar() {
		if !parser.IsQualifiedName(device) {
			return false
		}
		hasCDIdevice = true
	}

	for _, device := range i.DevicesFromMounts() {
		if !strings.HasPrefix(device, "cdi/") {
			return false
		}
		hasCDIdevice = true
	}
	return hasCDIdevice
}

// VisibleDevicesFromEnvVar returns the set of visible devices requested through
// the NVIDIA_VISIBLE_DEVICES environment variable.
func (i CUDA) VisibleDevicesFromEnvVar() []string {
	return i.DevicesFromEnvvars(EnvVarNvidiaVisibleDevices).List()
}

// VisibleDevicesFromMounts returns the set of visible devices requested as mounts.
func (i CUDA) VisibleDevicesFromMounts() []string {
	var devices []string
	for _, device := range i.DevicesFromMounts() {
		switch {
		case strings.HasPrefix(device, volumeMountDevicePrefixCDI):
			continue
		case strings.HasPrefix(device, volumeMountDevicePrefixImex):
			continue
		}
		devices = append(devices, device)
	}
	return devices
}

// DevicesFromMounts returns a list of device specified as mounts.
// TODO: This should be merged with getDevicesFromMounts used in the NVIDIA Container Runtime
func (i CUDA) DevicesFromMounts() []string {
	root := filepath.Clean(DeviceListAsVolumeMountsRoot)
	seen := make(map[string]bool)
	var devices []string
	for _, m := range i.mounts {
		source := filepath.Clean(m.Source)
		// Only consider mounts who's host volume is /dev/null
		if source != "/dev/null" {
			continue
		}

		destination := filepath.Clean(m.Destination)
		if seen[destination] {
			continue
		}
		seen[destination] = true

		// Only consider container mount points that begin with 'root'
		if !strings.HasPrefix(destination, root) {
			continue
		}

		// Grab the full path beyond 'root' and add it to the list of devices
		device := strings.Trim(strings.TrimPrefix(destination, root), "/")
		if len(device) == 0 {
			continue
		}
		devices = append(devices, device)
	}
	return devices
}

// CDIDevicesFromMounts returns a list of CDI devices specified as mounts on the image.
func (i CUDA) CDIDevicesFromMounts() []string {
	var devices []string
	for _, mountDevice := range i.DevicesFromMounts() {
		if !strings.HasPrefix(mountDevice, volumeMountDevicePrefixCDI) {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(mountDevice, volumeMountDevicePrefixCDI), "/", 3)
		if len(parts) != 3 {
			continue
		}
		vendor := parts[0]
		class := parts[1]
		device := parts[2]
		devices = append(devices, fmt.Sprintf("%s/%s=%s", vendor, class, device))
	}
	return devices
}

// ImexChannelsFromEnvVar returns the list of IMEX channels requested for the image.
func (i CUDA) ImexChannelsFromEnvVar() []string {
	imexChannels := i.DevicesFromEnvvars(EnvVarNvidiaImexChannels).List()
	if len(imexChannels) == 1 && imexChannels[0] == "all" {
		return nil
	}
	return imexChannels
}

// ImexChannelsFromMounts returns the list of IMEX channels requested for the image.
func (i CUDA) ImexChannelsFromMounts() []string {
	var channels []string
	for _, mountDevice := range i.DevicesFromMounts() {
		if !strings.HasPrefix(mountDevice, volumeMountDevicePrefixImex) {
			continue
		}
		channels = append(channels, strings.TrimPrefix(mountDevice, volumeMountDevicePrefixImex))
	}
	return channels
}
