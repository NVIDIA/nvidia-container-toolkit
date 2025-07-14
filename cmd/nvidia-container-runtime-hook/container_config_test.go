package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestGetNvidiaConfig(t *testing.T) {
	var tests = []struct {
		description    string
		env            map[string]string
		privileged     bool
		hookConfig     *HookConfig
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
				Devices:            []string{"all"},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"all"},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				envNVVisibleDevices: "void",
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
				Devices:            []string{""},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, no requirements",
			env: map[string]string{
				envCUDAVersion:          "9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "video,display",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{"cuda>=9.0"},
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, requirements set",
			env: map[string]string{
				envCUDAVersion:              "9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "video,display",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
			},
		},
		{
			description: "Legacy image, devices set, capabilities set, requirements set, disable requirements",
			env: map[string]string{
				envCUDAVersion:              "9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "video,display",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
				envNVDisableRequire:         "true",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{},
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
				Devices:            []string{"all"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				envNVVisibleDevices: "void",
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
				Devices:            []string{""},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: image.SupportedDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
			},
		},
		{
			description: "Modern image, devices set, capabilities set, no requirements",
			env: map[string]string{
				envNVRequireCUDA:        "cuda>=9.0",
				envNVVisibleDevices:     "gpu0,gpu1",
				envNVDriverCapabilities: "video,display",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{"cuda>=9.0"},
			},
		},
		{
			description: "Modern image, devices set, capabilities set, requirements set",
			env: map[string]string{
				envNVRequireCUDA:            "cuda>=9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "video,display",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{"cuda>=9.0", "req0=true", "req1=false"},
			},
		},
		{
			description: "Modern image, devices set, capabilities set, requirements set, disable requirements",
			env: map[string]string{
				envNVRequireCUDA:            "cuda>=9.0",
				envNVVisibleDevices:         "gpu0,gpu1",
				envNVDriverCapabilities:     "video,display",
				envNVRequirePrefix + "REQ0": "req0=true",
				envNVRequirePrefix + "REQ1": "req1=false",
				envNVDisableRequire:         "true",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{},
			},
		},
		{
			description: "No cuda envs, devices 'all'",
			env: map[string]string{
				envNVVisibleDevices: "all",
			},
			privileged: false,

			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{},
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
				Devices:            []string{"all"},
				MigConfigDevices:   "mig0,mig1",
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
				Devices:            []string{"all"},
				MigMonitorDevices:  "mig0,mig1",
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
				Requirements:       []string{"cuda>=9.0"},
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
		{
			description: "Hook config set as driver-capabilities-all",
			env: map[string]string{
				envNVVisibleDevices:     "all",
				envNVDriverCapabilities: "all",
			},
			privileged: true,
			hookConfig: &HookConfig{
				SupportedDriverCapabilities: "video,display",
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: "display,video",
			},
		},
		{
			description: "Hook config set, envvar sets driver-capabilities",
			env: map[string]string{
				envNVVisibleDevices:     "all",
				envNVDriverCapabilities: "video,display",
			},
			privileged: true,
			hookConfig: &HookConfig{
				SupportedDriverCapabilities: "video,display,compute,utility",
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: "display,video",
			},
		},
		{
			description: "Hook config set, envvar unset sets default driver-capabilities",
			env: map[string]string{
				envNVVisibleDevices: "all",
			},
			privileged: true,
			hookConfig: &HookConfig{
				SupportedDriverCapabilities: "video,display,utility,compute",
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
			},
		},
		{
			description: "Hook config set, swarmResource overrides device selection",
			env: map[string]string{
				envNVVisibleDevices:     "all",
				"DOCKER_SWARM_RESOURCE": "GPU1,GPU2",
			},
			privileged: true,
			hookConfig: &HookConfig{
				SwarmResource:               "DOCKER_SWARM_RESOURCE",
				SupportedDriverCapabilities: "video,display,utility,compute",
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"GPU1", "GPU2"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
			},
		},
		{
			description: "Hook config set, comma separated swarmResource is split and overrides device selection",
			env: map[string]string{
				envNVVisibleDevices:     "all",
				"DOCKER_SWARM_RESOURCE": "GPU1,GPU2",
			},
			privileged: true,
			hookConfig: &HookConfig{
				SwarmResource:               "NOT_DOCKER_SWARM_RESOURCE,DOCKER_SWARM_RESOURCE",
				SupportedDriverCapabilities: "video,display,utility,compute",
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"GPU1", "GPU2"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := image.New(
				image.WithEnvMap(tc.env),
			)
			// Wrap the call to getNvidiaConfig() in a closure.
			var config *nvidiaConfig
			getConfig := func() {
				hookConfig := tc.hookConfig
				if hookConfig == nil {
					defaultConfig, _ := getDefaultHookConfig()
					hookConfig = &defaultConfig
				}
				config = getNvidiaConfig(hookConfig, image, nil, tc.privileged)
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
		})
	}
}

func TestGetDevicesFromMounts(t *testing.T) {
	var tests = []struct {
		description     string
		mounts          []Mount
		expectedDevices []string
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
			expectedDevices: []string{"GPU0", "GPU1"},
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
			expectedDevices: []string{"GPU0-MIG0/0/1", "GPU1-MIG0/0/1"},
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
		expectedDevices    []string
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
			expectedDevices:    []string{"GPU0", "GPU1"},
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
			expectedDevices:    []string{"GPU0", "GPU1"},
		},
		{
			description:        "No mount devices, unprivileged, accept unprivileged",
			mountDevices:       nil,
			envvarDevices:      "GPU0,GPU1",
			privileged:         false,
			acceptUnprivileged: true,
			acceptMounts:       true,
			expectedDevices:    []string{"GPU0", "GPU1"},
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
			expectedDevices:    []string{"GPU2", "GPU3"},
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
			var devices []string
			getDevices := func() {
				image, _ := image.New(
					image.WithEnvMap(
						map[string]string{
							envNVVisibleDevices: tc.envvarDevices,
						},
					),
				)
				hookConfig, _ := getDefaultHookConfig()
				hookConfig.AcceptEnvvarUnprivileged = tc.acceptUnprivileged
				hookConfig.AcceptDeviceListAsVolumeMounts = tc.acceptMounts
				devices = getDevices(&hookConfig, image, tc.mountDevices, tc.privileged)
			}

			// For all other tests, just grab the devices and check the results
			getDevices()

			require.Equal(t, tc.expectedDevices, devices)
		})
	}
}

func TestGetDevicesFromEnvvar(t *testing.T) {
	envDockerResourceGPUs := "DOCKER_RESOURCE_GPUS"
	gpuID := "GPU-12345"
	anotherGPUID := "GPU-67890"
	thirdGPUID := "MIG-12345"

	var tests = []struct {
		description          string
		swarmResourceEnvvars []string
		env                  map[string]string
		expectedDevices      []string
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
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices: gpuID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				envNVVisibleDevices: gpuID,
				envCUDAVersion:      "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				envCUDAVersion: "legacy",
			},
			expectedDevices: []string{"all"},
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
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
				envCUDAVersion:        "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
				envCUDAVersion:        "legacy",
			},
			expectedDevices: []string{"all"},
		},
		// Add the `DOCKER_RESOURCE_GPUS` envvar and ensure that this is selected when
		// enabled
		{
			description:          "empty env returns nil for non-legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
		},
		{
			description:          "blank DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "",
			},
		},
		{
			description:          "'void' DOCKER_RESOURCE_GPUS returns nil for non-legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "void",
			},
		},
		{
			description:          "'none' DOCKER_RESOURCE_GPUS returns empty for non-legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: "none",
			},
			expectedDevices: []string{""},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS set returns value for non-legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS set returns value for legacy image",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: gpuID,
				envCUDAVersion:        "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS is selected if present",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS overrides NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{envDockerResourceGPUs},
			env: map[string]string{
				envNVVisibleDevices:   gpuID,
				envDockerResourceGPUs: anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS_ADDITIONAL overrides NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				envNVVisibleDevices:               gpuID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:          "All available swarm resource envvars are selected and override NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				envNVVisibleDevices:               gpuID,
				"DOCKER_RESOURCE_GPUS":            thirdGPUID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{thirdGPUID, anotherGPUID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS_ADDITIONAL or DOCKER_RESOURCE_GPUS override NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				envNVVisibleDevices:               gpuID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := image.New(
				image.WithEnvMap(tc.env),
			)
			devices := getDevicesFromEnvvar(image, tc.swarmResourceEnvvars)
			require.EqualValues(t, tc.expectedDevices, devices)
		})
	}
}

func TestGetDriverCapabilities(t *testing.T) {

	supportedCapabilities := "compute,display,utility,video"

	testCases := []struct {
		description           string
		env                   map[string]string
		legacyImage           bool
		supportedCapabilities string
		expectedPanic         bool
		expectedCapabilities  string
	}{
		{
			description: "Env is set for legacy image",
			env: map[string]string{
				envNVDriverCapabilities: "display,video",
			},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  "display,video",
		},
		{
			description: "Env is all for legacy image",
			env: map[string]string{
				envNVDriverCapabilities: "all",
			},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  supportedCapabilities,
		},
		{
			description: "Env is empty for legacy image",
			env: map[string]string{
				envNVDriverCapabilities: "",
			},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  image.DefaultDriverCapabilities.String(),
		},
		{
			description:           "Env unset for legacy image is 'all'",
			env:                   map[string]string{},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  supportedCapabilities,
		},
		{
			description: "Env is set for modern image",
			env: map[string]string{
				envNVDriverCapabilities: "display,video",
			},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  "display,video",
		},
		{
			description:           "Env unset for modern image is default",
			env:                   map[string]string{},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  image.DefaultDriverCapabilities.String(),
		},
		{
			description: "Env is all for modern image",
			env: map[string]string{
				envNVDriverCapabilities: "all",
			},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  supportedCapabilities,
		},
		{
			description: "Env is empty for modern image",
			env: map[string]string{
				envNVDriverCapabilities: "",
			},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  image.DefaultDriverCapabilities.String(),
		},
		{
			description: "Invalid capabilities panic",
			env: map[string]string{
				envNVDriverCapabilities: "compute,utility",
			},
			supportedCapabilities: "not-compute,not-utility",
			expectedPanic:         true,
		},
		{
			description:           "Default is restricted for modern image",
			legacyImage:           false,
			supportedCapabilities: "compute",
			expectedCapabilities:  "compute",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var capabilities string

			c := HookConfig{
				SupportedDriverCapabilities: tc.supportedCapabilities,
			}

			image, _ := image.New(
				image.WithEnvMap(tc.env),
			)
			getDriverCapabilities := func() {
				capabilities = c.getDriverCapabilities(image, tc.legacyImage).String()
			}

			if tc.expectedPanic {
				require.Panics(t, getDriverCapabilities)
				return
			}

			getDriverCapabilities()
			require.EqualValues(t, tc.expectedCapabilities, capabilities)
		})
	}
}
