package main

import (
	"testing"

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
				image.WithPrivileged(tc.privileged),
				image.WithPreferredVisibleDevicesEnvVars(tc.hookConfig.getSwarmResourceEnvvars()...),
			)

			// Wrap the call to getNvidiaConfig() in a closure.
			var cfg *nvidiaConfig
			getConfig := func() {
				hookCfg := tc.hookConfig
				if hookCfg == nil {
					defaultConfig, _ := config.GetDefault()
					hookCfg = &hookConfig{Config: defaultConfig}
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
