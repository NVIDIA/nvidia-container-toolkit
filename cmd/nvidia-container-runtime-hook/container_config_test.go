package main

import (
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestGetNvidiaConfig(t *testing.T) {
	var tests = []struct {
		description    string
		env            map[string]string
		privileged     bool
		hookConfig     *hookConfig
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
				image.EnvVarCudaVersion: "9.0",
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
				image.EnvVarCudaVersion:          "9.0",
				image.EnvVarNvidiaVisibleDevices: "all",
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
				image.EnvVarCudaVersion:          "9.0",
				image.EnvVarNvidiaVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Legacy image, devices 'void', no capabilities, no requirements",
			env: map[string]string{
				image.EnvVarCudaVersion:          "9.0",
				image.EnvVarNvidiaVisibleDevices: "void",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Legacy image, devices 'none', no capabilities, no requirements",
			env: map[string]string{
				image.EnvVarCudaVersion:          "9.0",
				image.EnvVarNvidiaVisibleDevices: "none",
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
				image.EnvVarCudaVersion:          "9.0",
				image.EnvVarNvidiaVisibleDevices: "gpu0,gpu1",
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
				image.EnvVarCudaVersion:              "9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "",
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
				image.EnvVarCudaVersion:              "9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "all",
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
				image.EnvVarCudaVersion:              "9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
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
				image.EnvVarCudaVersion:              "9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
				image.NvidiaRequirePrefix + "REQ0":   "req0=true",
				image.NvidiaRequirePrefix + "REQ1":   "req1=false",
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
				image.EnvVarCudaVersion:              "9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
				image.NvidiaRequirePrefix + "REQ0":   "req0=true",
				image.NvidiaRequirePrefix + "REQ1":   "req1=false",
				image.EnvVarNvidiaDisableRequire:     "true",
			},
			privileged: false,
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"gpu0", "gpu1"},
				DriverCapabilities: "display,video",
				Requirements:       []string{},
			},
		},
		{
			description: "Modern image, no devices, no capabilities, no requirements, no image.EnvVarCudaVersion",
			env: map[string]string{
				image.EnvVarNvidiaRequireCuda: "cuda>=9.0",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, no devices, no capabilities, no requirement, image.EnvVarCudaVersion set",
			env: map[string]string{
				image.EnvVarCudaVersion:       "9.0",
				image.EnvVarNvidiaRequireCuda: "cuda>=9.0",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'all', no capabilities, no requirements",
			env: map[string]string{
				image.EnvVarNvidiaRequireCuda:    "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices: "all",
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
				image.EnvVarNvidiaRequireCuda:    "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices: "",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'void', no capabilities, no requirements",
			env: map[string]string{
				image.EnvVarNvidiaRequireCuda:    "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices: "void",
			},
			privileged:     false,
			expectedConfig: nil,
		},
		{
			description: "Modern image, devices 'none', no capabilities, no requirements",
			env: map[string]string{
				image.EnvVarNvidiaRequireCuda:    "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices: "none",
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
				image.EnvVarNvidiaRequireCuda:    "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices: "gpu0,gpu1",
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
				image.EnvVarNvidiaRequireCuda:        "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "",
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
				image.EnvVarNvidiaRequireCuda:        "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "all",
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
				image.EnvVarNvidiaRequireCuda:        "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
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
				image.EnvVarNvidiaRequireCuda:        "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
				image.NvidiaRequirePrefix + "REQ0":   "req0=true",
				image.NvidiaRequirePrefix + "REQ1":   "req1=false",
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
				image.EnvVarNvidiaRequireCuda:        "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:     "gpu0,gpu1",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
				image.NvidiaRequirePrefix + "REQ0":   "req0=true",
				image.NvidiaRequirePrefix + "REQ1":   "req1=false",
				image.EnvVarNvidiaDisableRequire:     "true",
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
				image.EnvVarNvidiaVisibleDevices: "all",
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
				image.EnvVarNvidiaRequireCuda:      "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:   "all",
				image.EnvVarNvidiaMigConfigDevices: "mig0,mig1",
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
				image.EnvVarNvidiaRequireCuda:      "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:   "all",
				image.EnvVarNvidiaMigConfigDevices: "mig0,mig1",
			},
			privileged:    false,
			expectedPanic: true,
		},
		{
			description: "Modern image, devices 'all', migMonitor set, privileged",
			env: map[string]string{
				image.EnvVarNvidiaRequireCuda:       "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:    "all",
				image.EnvVarNvidiaMigMonitorDevices: "mig0,mig1",
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
				image.EnvVarNvidiaRequireCuda:       "cuda>=9.0",
				image.EnvVarNvidiaVisibleDevices:    "all",
				image.EnvVarNvidiaMigMonitorDevices: "mig0,mig1",
			},
			privileged:    false,
			expectedPanic: true,
		},
		{
			description: "Hook config set as driver-capabilities-all",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices:     "all",
				image.EnvVarNvidiaDriverCapabilities: "all",
			},
			privileged: true,
			hookConfig: &hookConfig{
				Config: &config.Config{
					SupportedDriverCapabilities: "video,display",
				},
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: "display,video",
			},
		},
		{
			description: "Hook config set, envvar sets driver-capabilities",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices:     "all",
				image.EnvVarNvidiaDriverCapabilities: "video,display",
			},
			privileged: true,
			hookConfig: &hookConfig{
				Config: &config.Config{
					SupportedDriverCapabilities: "video,display,compute,utility",
				},
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: "display,video",
			},
		},
		{
			description: "Hook config set, envvar unset sets default driver-capabilities",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "all",
			},
			privileged: true,
			hookConfig: &hookConfig{
				Config: &config.Config{
					SupportedDriverCapabilities: "video,display,utility,compute",
				},
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"all"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
			},
		},
		{
			description: "Hook config set, swarmResource overrides device selection",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "all",
				"DOCKER_SWARM_RESOURCE":          "GPU1,GPU2",
			},
			privileged: true,
			hookConfig: &hookConfig{
				Config: &config.Config{
					SwarmResource:               "DOCKER_SWARM_RESOURCE",
					SupportedDriverCapabilities: "video,display,utility,compute",
				},
			},
			expectedConfig: &nvidiaConfig{
				Devices:            []string{"GPU1", "GPU2"},
				DriverCapabilities: image.DefaultDriverCapabilities.String(),
			},
		},
		{
			description: "Hook config set, comma separated swarmResource is split and overrides device selection",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "all",
				"DOCKER_SWARM_RESOURCE":          "GPU1,GPU2",
			},
			privileged: true,
			hookConfig: &hookConfig{
				Config: &config.Config{
					SwarmResource:               "NOT_DOCKER_SWARM_RESOURCE,DOCKER_SWARM_RESOURCE",
					SupportedDriverCapabilities: "video,display,utility,compute",
				},
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
			var cfg *nvidiaConfig
			getConfig := func() {
				hookCfg := tc.hookConfig
				if hookCfg == nil {
					defaultConfig, _ := config.GetDefault()
					hookCfg = &hookConfig{defaultConfig}
				}
				cfg = hookCfg.getNvidiaConfig(image, tc.privileged)
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
				require.Nil(t, cfg, tc.description)
				return
			}

			require.NotNil(t, cfg, tc.description)

			require.Equal(t, tc.expectedConfig.Devices, cfg.Devices)
			require.Equal(t, tc.expectedConfig.MigConfigDevices, cfg.MigConfigDevices)
			require.Equal(t, tc.expectedConfig.MigMonitorDevices, cfg.MigMonitorDevices)
			require.Equal(t, tc.expectedConfig.DriverCapabilities, cfg.DriverCapabilities)

			require.ElementsMatch(t, tc.expectedConfig.Requirements, cfg.Requirements)
		})
	}
}

func TestDeviceListSourcePriority(t *testing.T) {
	var tests = []struct {
		description        string
		mountDevices       []specs.Mount
		envvarDevices      string
		privileged         bool
		acceptUnprivileged bool
		acceptMounts       bool
		expectedDevices    []string
	}{
		{
			description: "Mount devices, unprivileged, no accept unprivileged",
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU1"),
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
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU1"),
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
			mountDevices: []specs.Mount{
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU0"),
				},
				{
					Source:      "/dev/null",
					Destination: filepath.Join(image.DeviceListAsVolumeMountsRoot, "GPU1"),
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
							image.EnvVarNvidiaVisibleDevices: tc.envvarDevices,
						},
					),
					image.WithMounts(tc.mountDevices),
				)
				defaultConfig, _ := config.GetDefault()
				cfg := &hookConfig{defaultConfig}
				cfg.AcceptEnvvarUnprivileged = tc.acceptUnprivileged
				cfg.AcceptDeviceListAsVolumeMounts = tc.acceptMounts
				devices = cfg.getDevices(image, tc.privileged)
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
				image.EnvVarNvidiaVisibleDevices: "",
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "void",
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "none",
			},
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: gpuID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: gpuID,
				image.EnvVarCudaVersion:          "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				image.EnvVarCudaVersion: "legacy",
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
				image.EnvVarNvidiaVisibleDevices: "",
				envDockerResourceGPUs:            anotherGPUID,
			},
		},
		{
			description: "'void' NVIDIA_VISIBLE_DEVICES returns nil for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "void",
				envDockerResourceGPUs:            anotherGPUID,
			},
		},
		{
			description: "'none' NVIDIA_VISIBLE_DEVICES returns empty for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: "none",
				envDockerResourceGPUs:            anotherGPUID,
			},
			expectedDevices: []string{""},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for non-legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:            anotherGPUID,
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES set returns value for legacy image",
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:            anotherGPUID,
				image.EnvVarCudaVersion:          "legacy",
			},
			expectedDevices: []string{gpuID},
		},
		{
			description: "empty env returns all for legacy image",
			env: map[string]string{
				envDockerResourceGPUs:   anotherGPUID,
				image.EnvVarCudaVersion: "legacy",
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
				envDockerResourceGPUs:   gpuID,
				image.EnvVarCudaVersion: "legacy",
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
				image.EnvVarNvidiaVisibleDevices: gpuID,
				envDockerResourceGPUs:            anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS_ADDITIONAL overrides NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices:  gpuID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{anotherGPUID},
		},
		{
			description:          "All available swarm resource envvars are selected and override NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices:  gpuID,
				"DOCKER_RESOURCE_GPUS":            thirdGPUID,
				"DOCKER_RESOURCE_GPUS_ADDITIONAL": anotherGPUID,
			},
			expectedDevices: []string{thirdGPUID, anotherGPUID},
		},
		{
			description:          "DOCKER_RESOURCE_GPUS_ADDITIONAL or DOCKER_RESOURCE_GPUS override NVIDIA_VISIBLE_DEVICES if present",
			swarmResourceEnvvars: []string{"DOCKER_RESOURCE_GPUS", "DOCKER_RESOURCE_GPUS_ADDITIONAL"},
			env: map[string]string{
				image.EnvVarNvidiaVisibleDevices:  gpuID,
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
				image.EnvVarNvidiaDriverCapabilities: "display,video",
			},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  "display,video",
		},
		{
			description: "Env is all for legacy image",
			env: map[string]string{
				image.EnvVarNvidiaDriverCapabilities: "all",
			},
			legacyImage:           true,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  supportedCapabilities,
		},
		{
			description: "Env is empty for legacy image",
			env: map[string]string{
				image.EnvVarNvidiaDriverCapabilities: "",
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
				image.EnvVarNvidiaDriverCapabilities: "display,video",
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
				image.EnvVarNvidiaDriverCapabilities: "all",
			},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  supportedCapabilities,
		},
		{
			description: "Env is empty for modern image",
			env: map[string]string{
				image.EnvVarNvidiaDriverCapabilities: "",
			},
			legacyImage:           false,
			supportedCapabilities: supportedCapabilities,
			expectedCapabilities:  image.DefaultDriverCapabilities.String(),
		},
		{
			description: "Invalid capabilities panic",
			env: map[string]string{
				image.EnvVarNvidiaDriverCapabilities: "compute,utility",
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

			c := hookConfig{
				Config: &config.Config{
					SupportedDriverCapabilities: tc.supportedCapabilities,
				},
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
