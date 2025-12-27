/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package modifier

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	visibleDevicesEnvvar = "NVIDIA_VISIBLE_DEVICES"
	visibleDevicesVoid   = "void"
	visibleDevicesNone   = "none"
	visibleDevicesAll    = "all"
)

// NewStableRuntimeModifier creates an OCI spec modifier that inserts the NVIDIA Container Runtime Hook into an OCI
// spec. The specified logger is used to capture log output.
func NewStableRuntimeModifier(logger logger.Interface, nvidiaContainerRuntimeHookPath string) oci.SpecModifier {
	m := stableRuntimeModifier{
		logger:                         logger,
		nvidiaContainerRuntimeHookPath: nvidiaContainerRuntimeHookPath,
	}

	return &m
}

// stableRuntimeModifier modifies an OCI spec inplace, inserting the nvidia-container-runtime-hook as a
// prestart hook. If the hook is already present, no modification is made.
type stableRuntimeModifier struct {
	logger                         logger.Interface
	nvidiaContainerRuntimeHookPath string
}

// Modify applies the required modification to the incoming OCI spec, inserting the nvidia-container-runtime-hook
// as a prestart hook.
func (m stableRuntimeModifier) Modify(spec *specs.Spec) error {
	// If an NVIDIA Container Runtime Hook already exists, we don't make any modifications to the spec.
	if spec.Hooks != nil {
		for _, hook := range spec.Hooks.Prestart {
			hook := hook
			if isNVIDIAContainerRuntimeHook(&hook) {
				m.logger.Infof("Existing nvidia prestart hook (%v) found in OCI spec", hook.Path)
				return nil
			}
		}
	}

	path := m.nvidiaContainerRuntimeHookPath
	m.logger.Infof("Using prestart hook path: %v", path)
	args := []string{filepath.Base(path)}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	}
	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	if err := m.addDeviceCgroupRules(spec); err != nil {
		return err
	}

	return nil
}

// addDeviceCgroupRules adds cgroup device rules based on NVIDIA_VISIBLE_DEVICES
func (m stableRuntimeModifier) addDeviceCgroupRules(spec *specs.Spec) error {
	// Initialize Linux.Resources if needed
	if spec.Linux == nil {
		spec.Linux = &specs.Linux{}
	}
	if spec.Linux.Resources == nil {
		spec.Linux.Resources = &specs.LinuxResources{}
	}

	// Get NVIDIA_VISIBLE_DEVICES from container env
	visibleDevices := m.getEnvVar(spec, visibleDevicesEnvvar)

	// Skip if void or none
	if visibleDevices == "" || visibleDevices == visibleDevicesVoid || visibleDevices == visibleDevicesNone {
		m.logger.Warning("NVIDIA_VISIBLE_DEVICES is void/none/empty, skipping cgroup rules")
		return nil
	}

	// Add common control devices (nvidiactl, nvidia-uvm, etc.)
	if err := m.addCommonDevices(spec); err != nil {
		return fmt.Errorf("failed to add common devices: %v", err)
	}

	// Add GPU-specific devices based on NVIDIA_VISIBLE_DEVICES
	if err := m.addGPUDevices(spec, visibleDevices); err != nil {
		return fmt.Errorf("failed to add GPU devices: %v", err)
	}

	return nil
}

func (m stableRuntimeModifier) getEnvVar(spec *specs.Spec, key string) string {
	if spec.Process == nil {
		return ""
	}

	prefix := key + "="
	for _, env := range spec.Process.Env {
		if strings.HasPrefix(env, prefix) {
			return strings.TrimPrefix(env, prefix)
		}
	}
	return ""
}

func (m stableRuntimeModifier) addCommonDevices(spec *specs.Spec) error {
	commonDevices := []string{
		"/dev/nvidiactl",
		"/dev/nvidia-uvm",
		"/dev/nvidia-uvm-tools",
		"/dev/nvidia-modeset",
	}

	for _, devicePath := range commonDevices {
		rule, err := m.devicePathToRule(devicePath)
		if err != nil {
			return fmt.Errorf("failed to add common device %s: %v", devicePath, err)
		}
		spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, *rule)
		m.logger.Debugf("Added cgroup rule for %s (major=%d, minor=%d)",
			devicePath, *rule.Major, *rule.Minor)
	}

	return nil
}

func (m stableRuntimeModifier) addGPUDevices(spec *specs.Spec, visibleDevices string) error {
	deviceList := strings.Split(visibleDevices, ",")

	for _, device := range deviceList {
		device = strings.TrimSpace(device)
		if device == "" {
			continue
		}

		if device == visibleDevicesAll {
			return m.addAllGPUDevices(spec)
		}

		devicePaths, err := m.resolveDevicePaths(device)
		if err != nil {
			return fmt.Errorf("failed to resolve device %s: %v", device, err)
		}

		for _, devicePath := range devicePaths {
			rule, err := m.devicePathToRule(devicePath)
			if err != nil {
				return fmt.Errorf("failed to add device %s: %v", devicePath, err)
			}
			spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, *rule)
			m.logger.Debugf("Added cgroup rule for %s (major=%d, minor=%d)",
				devicePath, *rule.Major, *rule.Minor)
		}
	}

	return nil
}

func (m stableRuntimeModifier) addAllGPUDevices(spec *specs.Spec) error {
	// Find all /dev/nvidia[0-9]* devices
	matches, err := filepath.Glob("/dev/nvidia[0-9]*")
	if err != nil {
		return fmt.Errorf("failed to glob nvidia devices: %v", err)
	}

	for _, devicePath := range matches {
		rule, err := m.devicePathToRule(devicePath)
		if err != nil {
			return fmt.Errorf("failed to add device %s: %v", devicePath, err)
		}
		spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, *rule)
		m.logger.Debugf("Added cgroup rule for %s (major=%d, minor=%d)",
			devicePath, *rule.Major, *rule.Minor)
	}

	// Also add nvidia-caps for MIG support
	capsMatches, _ := filepath.Glob("/dev/nvidia-caps/nvidia-cap[0-9]*")
	for _, devicePath := range capsMatches {
		rule, err := m.devicePathToRule(devicePath)
		if err != nil {
			return fmt.Errorf("failed to add device %s: %v", devicePath, err)
		}
		spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, *rule)
	}

	return nil
}

func (m stableRuntimeModifier) resolveDevicePaths(device string) ([]string, error) {
	var paths []string

	if idx, err := strconv.Atoi(device); err == nil {
		paths = append(paths, fmt.Sprintf("/dev/nvidia%d", idx))
		return paths, nil
	}

	return nil, fmt.Errorf("unknown device format: %s", device)
}

func (m stableRuntimeModifier) devicePathToRule(path string) (*specs.LinuxDeviceCgroup, error) {
	var stat unix.Stat_t
	if err := unix.Stat(path, &stat); err != nil {
		return nil, fmt.Errorf("stat %s: %v", path, err)
	}

	// Verify it's a character device
	if stat.Mode&unix.S_IFCHR == 0 {
		return nil, fmt.Errorf("%s is not a character device", path)
	}

	major := int64(unix.Major(stat.Rdev))
	minor := int64(unix.Minor(stat.Rdev))

	return &specs.LinuxDeviceCgroup{
		Allow:  true,
		Type:   "c",
		Major:  &major,
		Minor:  &minor,
		Access: "rwm",
	}, nil
}
