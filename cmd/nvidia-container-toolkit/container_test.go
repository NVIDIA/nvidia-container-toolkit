package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetNvidiaConfig(t *testing.T) {
	var tests = []struct {
		description    string
		env            map[string]string
		privileged     bool
		expectedConfig *nvidiaConfig
		expectedPanic  bool
	}{
		{
			description:    "No environment, unprivileged",
			env:            map[string]string{},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description:    "No environment, privileged",
			env:            map[string]string{},
			privileged:     true,
			expectedConfig: nil,
		},
		{
			description: "Legacy image, no devices, no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion: "9.0",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices 'all', no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion:      "9.0",
				envNVVisibleDevices: "all",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices 'empty', no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion:      "9.0",
				envNVVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Legacy image, devices 'void', no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion:      "9.0",
				envNVVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Legacy image, devices 'none', no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion:      "9.0",
				envNVVisibleDevices: "none",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, no capabilities, no requirements",
			env: map[string]string{
				envCUDAVersion:      "9.0",
				envNVVisibleDevices: "gpu0,gpu1",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, capabilities 'empty', no requirements",
			env: map[string]string{
				envCUDAVersion:          "9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, capabilities 'all', no requirements",
			env: map[string]string{
				envCUDAVersion:          "9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "all",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, no requirements",
			env: map[string]string{
				envCUDAVersion:          "9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "cap0,cap1",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, requirements set",
			env: map[string]string{
				envCUDAVersion:              "9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "cap0,cap1",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
				DisableRequire:     false,
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, requirements set, disable requirements",
			env: map[string]string{
				envCUDAVersion:              "9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "cap0,cap1",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
				envNVDisableRequire:         "true",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
				DisableRequire:     true,
			},
		},
		{
			description: "Modern image, no devices, no capabilities, no requirements, no envCUDAVersion",
			env: map[string]string{
				envNVRequireCUDA: "cuda>=9.0",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, no devices, no capabilities, no requirement, envCUDAVersion set",
			env: map[string]string{
				envCUDAVersion:   "9.0",
				envNVRequireCUDA: "cuda>=9.0",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'all', no capabilities, no requirements",
			env: map[string]string{
				envNVRequireCUDA:    "cuda>=9.0",
				envNVVisibleDevices: "all",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices 'empty', no capabilities, no requirements",
			env: map[string]string{
				envNVRequireCUDA:    "cuda>=9.0",
				envNVVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'void', no capabilities, no requirements",
			env: map[string]string{
				envNVRequireCUDA:    "cuda>=9.0",
				envNVVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'none', no capabilities, no requirements",
			env: map[string]string{
				envNVRequireCUDA:    "cuda>=9.0",
				envNVVisibleDevices: "none",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, no capabilities, no requirements",
			env: map[string]string{
				envNVRequireCUDA:    "cuda>=9.0",
				envNVVisibleDevices: "gpu0,gpu1",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, capabilities 'empty', no requirements",
			env: map[string]string{
				envNVRequireCUDA:        "cuda>=9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, capabilities 'all', no requirements",
			env: map[string]string{
				envNVRequireCUDA:        "cuda>=9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "all",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: allDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, capabilities set, no requirements",
			env: map[string]string{
				envNVRequireCUDA:        "cuda>=9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "cap0,cap1",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, capabilities set, requirements set",
			env: map[string]string{
				envNVRequireCUDA:            "cuda>=9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "cap0,cap1",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices set, capabilities set, requirements set, disable requirements",
			env: map[string]string{
				envNVRequireCUDA:            "cuda>=9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "cap0,cap1",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
				envNVDisableRequire:         "true",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            "gpu0,gpu1",
				DriverCapabilities: "cap0,cap1",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
				DisableRequire:     true,
			},
		},
		{
			description: "No cuda envs, devices 'all'",
			env: map[string]string{
				envNVVisibleDevices: "all",
			},
			privileged: false,

			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices 'all', migConfig set, privileged",
			env: map[string]string{
				envNVRequireCUDA:      "cuda>=9.0",
				envNVVisibleDevices:   "all",
				envNVMigConfigDevices: "mig0,mig1",
			},
			privileged: true,
			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				MigConfigDevices:   "mig0,mig1",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices 'all', migConfig set, unprivileged",
			env: map[string]string{
				envNVRequireCUDA:      "cuda>=9.0",
				envNVVisibleDevices:   "all",
				envNVMigConfigDevices: "mig0,mig1",
			},
			privileged:    false,
			expectedPanic: true,
		},
		{
			description: "Modern image, devices 'all', migMonitor set, privileged",
			env: map[string]string{
				envNVRequireCUDA:       "cuda>=9.0",
				envNVVisibleDevices:    "all",
				envNVMigMonitorDevices: "mig0,mig1",
			},
			privileged: true,
			expectedConfig: &nvidiaConfig{
				Devices:            "all",
				MigMonitorDevices:  "mig0,mig1",
				DriverCapabilities: defaultDriverCapabilities,
				Requirements:       []string{"cuda>=9.0"},
				DisableRequire:     false,
			},
		},
		{
			description: "Modern image, devices 'all', migMonitor set, unprivileged",
			env: map[string]string{
				envNVRequireCUDA:       "cuda>=9.0",
				envNVVisibleDevices:    "all",
				envNVMigMonitorDevices: "mig0,mig1",
			},
			privileged:    false,
			expectedPanic: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			// Wrap the call to getNvidiaConfig() in a closure.
			var config *nvidiaConfig
			getConfig := func() {
				hookConfig := getDefaultHookConfig()
				config = getNvidiaConfig(&hookConfig, tc.env, nil, tc.privileged)
			}

			// For any tests that are expected to panic, make sure they do.
			if tc.expectedPanic {
				require.Panics(t, getConfig)
				return
			}

			// For all other tests, just grab the config
			getConfig()

			// And start comparing the test results to the expected results.
			if tc.expectedConfig == nil {
				require.Nil(t, config, tc.description)
				return
			}

			require.NotNil(t, config, tc.description)

			require.Equal(t, tc.expectedConfig.Devices, config.Devices)
			require.Equal(t, tc.expectedConfig.MigConfigDevices, config.MigConfigDevices)
			require.Equal(t, tc.expectedConfig.MigMonitorDevices, config.MigMonitorDevices)
			require.Equal(t, tc.expectedConfig.DriverCapabilities, config.DriverCapabilities)

			require.ElementsMatch(t, tc.expectedConfig.Requirements, config.Requirements)
			require.Equal(t, tc.expectedConfig.DisableRequire, config.DisableRequire)
		})
	}
}

func TestGetDevicesFromMounts(t *testing.T) {
	var tests = []struct {
		description     string
		mounts          []Mount
		expectedDevices *string
	}{
		{
			description:     "No mounts",
			mounts:          nil,
			expectedDevices: nil,
		},
		{
			description: "Host path is not /dev/null",
			mounts: []Mount{
				{
					Source:      "/not/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0"),
				},
			},
			expectedDevices: nil,
		},
		{
			description: "Container path is not prefixed by 'root'",
			mounts: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join("/other/prefix", "GPU0"),
				},
			},
			expectedDevices: nil,
		},
		{
			description: "Container path is only 'root'",
			mounts: []Mount{
				{
					Source:      "/dev/null",
					Destination: deviceListAsVolumeMountsRoot,
				},
			},
			expectedDevices: nil,
		},
		{
			description: "Discover 2 devices",
			mounts: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			expectedDevices: &[]string{"GPU0,GPU1"}[0],
		},
		{
			description: "Discover 2 devices with slashes in the name",
			mounts: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0-MIG0/0/1"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU1-MIG0/0/1"),
				},
			},
			expectedDevices: &[]string{"GPU0-MIG0/0/1,GPU1-MIG0/0/1"}[0],
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			devices := getDevicesFromMounts(tc.mounts)
			require.Equal(t, tc.expectedDevices, devices)
		})
	}
}

func TestDeviceListSourcePriority(t *testing.T) {
	var tests = []struct {
		description        string
		mountDevices       []Mount
		envvarDevices      string
		privileged         bool
		acceptUnprivileged bool
		acceptMounts       bool
		expectedDevices    *string
	}{
		{
			description: "Mount devices, unprivileged, no accept unprivileged",
			mountDevices: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    &[]string{"GPU0,GPU1"}[0],
		},
		{
			description:        "No mount devices, unprivileged, no accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    nil,
		},
		{
			description:        "No mount devices, privileged, no accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         true,
			acceptUnprivileged: false,
			acceptMounts:       true,
			expectedDevices:    &[]string{"GPU0,GPU1"}[0],
		},
		{
			description:        "No mount devices, unprivileged, accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         false,
			acceptUnprivileged: true,
			acceptMounts:       true,
			expectedDevices:    &[]string{"GPU0,GPU1"}[0],
		},
		{
			description: "Mount devices, unprivileged, accept unprivileged, no accept mounts",
			mountDevices: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: true,
			acceptMounts:       false,
			expectedDevices:    &[]string{"GPU2,GPU3"}[0],
		},
		{
			description: "Mount devices, unprivileged, no accept unprivileged, no accept mounts",
			mountDevices: []Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(deviceListAsVolumeMountsRoot, "GPU1"),
				},
			},
			envvarDevices:      "GPU2,GPU3",
			privileged:         false,
			acceptUnprivileged: false,
			acceptMounts:       false,
			expectedDevices:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			// Wrap the call to getDevices() in a closure.
			var devices *string
			getDevices := func() {
				env := map[string]string{
					envNVVisibleDevices: tc.envvarDevices,
				}
				hookConfig := getDefaultHookConfig()
				hookConfig.AcceptEnvvarUnprivileged = tc.acceptUnprivileged
				hookConfig.AcceptDeviceListAsVolumeMounts = tc.acceptMounts
				devices = getDevices(&hookConfig, env, tc.mountDevices, tc.privileged, false)
			}

			// For all other tests, just grab the devices and check the results
			getDevices()

			require.Equal(t, tc.expectedDevices, devices)
		})
	}
}

func TestGetDevicesFromEnvvar(t *testing.T) {
	all := "all"
	empty := ""
	envDockerResourceGPUs := "DOCKER_RESOURCE_GPUS"
	gpuID := "GPU-12345"
	anotherGPUID := "GPU-67890"

	var tests = []struct {
		description     string
		envSwarmGPU     *string
		env             map[string]string
		legacyImage     bool
		expectedDevices *string
	}{
		{
			description: "empty env returns nil for non-legacy image",
		},
		{
			description: "blank NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices: "",
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices: "void",
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices: "none",
			},
			expectedDevices: &empty,
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices: gpuID,
			},
			expectedDevices: &gpuID,
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				envNVVisibleDevices: gpuID,
			},
			legacyImage:     true,
			expectedDevices: &gpuID,
		},
		{
			description:     "empty env returns all for legacy image",
			legacyImage:     true,
			expectedDevices: &all,
		},
		// Add the `DOCKER_RESOURCE_GPUS` envvar and ensure that this is ignored when
		// not enabled
		{
			description: "missing NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
		},
		{
			description: "blank NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices:   "",
				envDockerResourceGPUs: anotherGPUID,
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices:   "void",
				envDockerResourceGPUs: anotherGPUID,
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices:   "none",
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: &empty,
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: &gpuID,
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
			},
			legacyImage:     true,
			expectedDevices: &gpuID,
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
			legacyImage:     true,
			expectedDevices: &all,
		},
		// Add the `DOCKER_RESOURCE_GPUS` envvar and ensure that this is selected when
		// enabled
		{
			description: "empty env returns nil for non-legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
		},
		{
			description: "blank DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: "",
			},
		},
		{
			description: "'void' DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: "void",
			},
		},
		{
			description: "'none' DOCKER_RESOURCE_GPUS returns empty for non-legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: "none",
			},
			expectedDevices: &empty,
		},
		{
			description: "DOCKER_RESOURCE_GPUS set returns value for non-legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
			},
			expectedDevices: &gpuID,
		},
		{
			description: "DOCKER_RESOURCE_GPUS set returns value for legacy image",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
			},
			legacyImage:     true,
			expectedDevices: &gpuID,
		},
		{
			description: "DOCKER_RESOURCE_GPUS is selected if present",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: &anotherGPUID,
		},
		{
			description: "DOCKER_RESOURCE_GPUS overrides NVIDIA_VISIBLE_DEVICES if present",
			envSwarmGPU: &envDockerResourceGPUs,
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: &anotherGPUID,
		},
	}

	for i, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			envSwarmGPU = tc.envSwarmGPU
			devices := getDevicesFromEnvvar(tc.env, tc.legacyImage)
			if tc.expectedDevices == nil {
				require.Nil(t, devices, "%d: %v", i, tc)
				return
			}

			require.NotNil(t, devices, "%d: %v", i, tc)
			require.Equal(t, *tc.expectedDevices, *devices, "%d: %v", i, tc)
		})
	}
}
