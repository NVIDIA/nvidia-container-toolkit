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
	"slices"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/mod/semver"
	"tags.cncf.io/container-device-interface/pkg/parser"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
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
	logger logger.Interface

	annotations  map[string]string
	env          map[string]string
	isPrivileged bool
	mounts       []specs.Mount

	annotationsPrefixes            []string
	acceptDeviceListAsVolumeMounts bool
	acceptEnvvarUnprivileged       bool
	ignoreImexChannelRequests      bool
	preferredVisibleDeviceEnvVars  []string
}

// NewCUDAImageFromSpec creates a CUDA image from the input OCI runtime spec.
// The process environment is read (if present) to construc the CUDA Image.
func NewCUDAImageFromSpec(spec *specs.Spec, opts ...Option) (CUDA, error) {
	if spec == nil {
		return New(opts...)
	}

	var env []string
	if spec.Process != nil {
		env = spec.Process.Env
	}

	specOpts := []Option{
		WithAnnotations(spec.Annotations),
		WithEnv(env),
		WithMounts(spec.Mounts),
		WithPrivileged(IsPrivileged((*OCISpec)(spec))),
	}

	return New(append(opts, specOpts...)...)
}

// newCUDAImageFromEnv creates a CUDA image from the input environment. The environment
// is a list of strings of the form ENVAR=VALUE.
func newCUDAImageFromEnv(env []string) (CUDA, error) {
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

func (i CUDA) IsPrivileged() bool {
	return i.isPrivileged
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

// devicesFromEnvvars returns the devices requested by the image through environment variables
func (i CUDA) devicesFromEnvvars(envVars ...string) []string {
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
		devices = []string{"all"}
	}

	// Environment variable unset or empty or "void": return nil
	if len(devices) == 0 || requested["void"] {
		devices = []string{"void"}
	}

	return NewVisibleDevices(devices...).List()
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
	for _, device := range i.VisibleDevices() {
		if !parser.IsQualifiedName(device) {
			return false
		}
		hasCDIdevice = true
	}
	return hasCDIdevice
}

// visibleEnvVars returns the environment variables that are used to determine device visibility.
// It returns the preferred environment variables that are set, or NVIDIA_VISIBLE_DEVICES if none are set.
func (i CUDA) visibleEnvVars() []string {
	var envVars []string
	for _, envVar := range i.preferredVisibleDeviceEnvVars {
		if !i.HasEnvvar(envVar) {
			continue
		}
		envVars = append(envVars, envVar)
	}
	if len(envVars) > 0 {
		return envVars
	}
	return []string{EnvVarNvidiaVisibleDevices}
}

// VisibleDevices returns a list of devices requested in the container image.
// If volume mount requests are enabled these are returned if requested,
// otherwise device requests through environment variables are considered.
// In cases where environment variable requests required privileged containers,
// such devices requests are ignored.
func (i CUDA) VisibleDevices() []string {
	// If annotation device requests are present, these are preferred.
	annotationDeviceRequests := i.cdiDeviceRequestsFromAnnotations()
	if len(annotationDeviceRequests) > 0 {
		return annotationDeviceRequests
	}

	// If enabled, try and get the device list from volume mounts first
	if i.acceptDeviceListAsVolumeMounts {
		volumeMountDeviceRequests := i.visibleDevicesFromMounts()
		if len(volumeMountDeviceRequests) > 0 {
			return volumeMountDeviceRequests
		}
	}

	// Get the Fallback to reading from the environment variable if privileges are correct
	envVarDeviceRequests := i.visibleDevicesFromEnvVar()
	if len(envVarDeviceRequests) == 0 {
		return nil
	}

	// If the container is privileged, or environment variable requests are
	// allowed for unprivileged containers, these devices are returned.
	if i.isPrivileged || i.acceptEnvvarUnprivileged {
		return envVarDeviceRequests
	}

	// We log a warning if we are ignoring the environment variable requests.
	envVars := i.visibleEnvVars()
	if len(envVars) > 0 {
		i.logger.Warningf("Ignoring devices requested by environment variable(s) in unprivileged container: %v", envVars)
	}

	return nil
}

// cdiDeviceRequestsFromAnnotations returns a list of devices specified in the
// annotations.
// Keys starting with the specified prefixes are considered and expected to
// contain a comma-separated list of fully-qualified CDI devices names.
// The format of the requested devices is not checked and the list is not
// deduplicated.
func (i CUDA) cdiDeviceRequestsFromAnnotations() []string {
	if len(i.annotationsPrefixes) == 0 || len(i.annotations) == 0 {
		return nil
	}

	var annotationKeys []string
	for key := range i.annotations {
		for _, prefix := range i.annotationsPrefixes {
			if strings.HasPrefix(key, prefix) {
				annotationKeys = append(annotationKeys, key)
				// There is no need to check additional prefixes since we
				// typically deduplicate devices in any case.
				break
			}
		}
	}
	// We sort the annotationKeys for consistent results.
	slices.Sort(annotationKeys)

	var devices []string
	for _, key := range annotationKeys {
		devices = append(devices, strings.Split(i.annotations[key], ",")...)
	}
	return devices
}

// visibleDevicesFromEnvVar returns the set of visible devices requested through environment variables.
// If any of the preferredVisibleDeviceEnvVars are present in the image, they
// are used to determine the visible devices. If this is not the case, the
// NVIDIA_VISIBLE_DEVICES environment variable is used.
func (i CUDA) visibleDevicesFromEnvVar() []string {
	envVars := i.visibleEnvVars()
	return i.devicesFromEnvvars(envVars...)
}

// visibleDevicesFromMounts returns the set of visible devices requested as mounts.
func (i CUDA) visibleDevicesFromMounts() []string {
	var devices []string
	for _, device := range i.requestsFromMounts() {
		switch {
		case strings.HasPrefix(device, volumeMountDevicePrefixImex):
			continue
		case strings.HasPrefix(device, volumeMountDevicePrefixCDI):
			name, err := cdiDeviceMountRequest(device).qualifiedName()
			if err != nil {
				i.logger.Warningf("Ignoring invalid mount request for CDI device %v: %v", device, err)
				continue
			}
			devices = append(devices, name)
		default:
			devices = append(devices, device)
		}

	}
	return devices
}

// requestsFromMounts returns a list of device specified as mounts.
func (i CUDA) requestsFromMounts() []string {
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

// a cdiDeviceMountRequest represents a CDI device requests as a mount.
// Here the host path /dev/null is mounted to a particular path in the container.
// The container path has the form:
// /var/run/nvidia-container-devices/cdi/<vendor>/<class>/<device>
// or
// /var/run/nvidia-container-devices/cdi/<vendor>/<class>=<device>
type cdiDeviceMountRequest string

// qualifiedName returns the fully-qualified name of the CDI device.
func (m cdiDeviceMountRequest) qualifiedName() (string, error) {
	if !strings.HasPrefix(string(m), volumeMountDevicePrefixCDI) {
		return "", fmt.Errorf("invalid mount CDI device request: %s", m)
	}

	requestedDevice := strings.TrimPrefix(string(m), volumeMountDevicePrefixCDI)
	if parser.IsQualifiedName(requestedDevice) {
		return requestedDevice, nil
	}

	parts := strings.SplitN(requestedDevice, "/", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid mount CDI device request: %s", m)
	}
	return fmt.Sprintf("%s/%s=%s", parts[0], parts[1], parts[2]), nil
}

func (i CUDA) ImexChannelRequests() []string {
	if i.ignoreImexChannelRequests {
		return nil
	}

	// If enabled, try and get the device list from volume mounts first
	if i.acceptDeviceListAsVolumeMounts {
		volumeMountDeviceRequests := i.imexChannelsFromMounts()
		if len(volumeMountDeviceRequests) > 0 {
			return volumeMountDeviceRequests
		}
	}

	// Get the Fallback to reading from the environment variable if privileges are correct
	envVarDeviceRequests := i.imexChannelsFromEnvVar()
	if len(envVarDeviceRequests) == 0 {
		return nil
	}

	// If the container is privileged, or environment variable requests are
	// allowed for unprivileged containers, these devices are returned.
	if i.isPrivileged || i.acceptEnvvarUnprivileged {
		return envVarDeviceRequests
	}

	// We log a warning if we are ignoring the environment variable requests.
	envVars := []string{EnvVarNvidiaImexChannels}
	if len(envVars) > 0 {
		i.logger.Warningf("Ignoring request by environment variable(s) in unprivileged container: %v", envVars)
	}

	return nil
}

// imexChannelsFromEnvVar returns the list of IMEX channels requested for the image.
func (i CUDA) imexChannelsFromEnvVar() []string {
	imexChannels := i.devicesFromEnvvars(EnvVarNvidiaImexChannels)
	if len(imexChannels) == 1 && imexChannels[0] == "all" {
		return nil
	}
	return imexChannels
}

// imexChannelsFromMounts returns the list of IMEX channels requested for the image.
func (i CUDA) imexChannelsFromMounts() []string {
	var channels []string
	for _, mountDevice := range i.requestsFromMounts() {
		if !strings.HasPrefix(mountDevice, volumeMountDevicePrefixImex) {
			continue
		}
		channels = append(channels, strings.TrimPrefix(mountDevice, volumeMountDevicePrefixImex))
	}
	return channels
}
