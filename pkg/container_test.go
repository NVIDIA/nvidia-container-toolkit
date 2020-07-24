package main

import (
	"reflect"
	"testing"
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
				config = getNvidiaConfig(tc.env, tc.privileged)
			}

			// For any tests that are expected to panic, make sure they do.
			if tc.expectedPanic {
				mustPanic(t, getConfig)
				return
			}

			// For all other tests, just grab the config
			getConfig()

			// And start comparing the test results to the expected results.
			if config == nil && tc.expectedConfig == nil {
				return
			}
			if config != nil && tc.expectedConfig != nil {
				if !reflect.DeepEqual(config.Devices, tc.expectedConfig.Devices) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				if !reflect.DeepEqual(config.MigConfigDevices, tc.expectedConfig.MigConfigDevices) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				if !reflect.DeepEqual(config.MigMonitorDevices, tc.expectedConfig.MigMonitorDevices) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				if !reflect.DeepEqual(config.DriverCapabilities, tc.expectedConfig.DriverCapabilities) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				if !elementsMatch(config.Requirements, tc.expectedConfig.Requirements) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				if !reflect.DeepEqual(config.DisableRequire, tc.expectedConfig.DisableRequire) {
					t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
				}
				return
			}
			t.Errorf("Unexpected nvidiaConfig (got: %v, wanted: %v)", config, tc.expectedConfig)
		})
	}
}

func elementsMatch(slice0, slice1 []string) bool {
	map0 := make(map[string]int)
	map1 := make(map[string]int)

	for _, e := range slice0 {
		map0[e]++
	}

	for _, e := range slice1 {
		map1[e]++
	}

	for k0, v0 := range map0 {
		if map1[k0] != v0 {
			return false
		}
	}

	for k1, v1 := range map1 {
		if map0[k1] != v1 {
			return false
		}
	}

	return true
}
