package main

import (
	"github.com/stretchr/testify/require"
	"sort"
	"strings"
	"testing"
)

func TestMergeVisibleDevicesEnvvars(t *testing.T) {
	var tests = []struct {
		name        string
		input       []string
		expected    string
		enableMerge bool
	}{
		{
			"Simple Merge Enabled",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"0,1,2,3,4,5",
			true,
		},
		{
			"Simple Merge Disabled",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"",
			false,
		},
		{
			"Merge No Override (Enabled)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES=all",
			},
			"all",
			true,
		},
		{
			"Merge No Override (Disabled)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES=all",
			},
			"all",
			false,
		},
		{
			"Merge Override (Enabled, Before)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES=all",
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"0,1,2,3,4,5",
			true,
		},
		{
			"Merge Override (Enabled, After)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
				"NVIDIA_VISIBLE_DEVICES=all",
			},
			"0,1,2,3,4,5",
			true,
		},
		{
			"Merge Override (Enabled, In Between)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES=all",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"0,1,2,3,4,5",
			true,
		},
		{
			"Merge Override (Disabled, Before)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES=all",
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"all",
			false,
		},
		{
			"Merge Override (Disabled, After)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
				"NVIDIA_VISIBLE_DEVICES=all",
			},
			"all",
			false,
		},
		{
			"Merge Override (Disabled, In Between)",
			[]string{
				"NVIDIA_VISIBLE_DEVICES_0=0,1",
				"NVIDIA_VISIBLE_DEVICES_1=2,3",
				"NVIDIA_VISIBLE_DEVICES=all",
				"NVIDIA_VISIBLE_DEVICES_WHATEVER=4,5",
			},
			"all",
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := CLIConfig{
				AlphaMergeVisibleDevicesEnvvars: tc.enableMerge,
			}
			envvars := getEnvMap(tc.input, config)
			devices := strings.Split(envvars[envNVVisibleDevices], ",")
			sort.Strings(devices)
			require.Equal(t, tc.expected, strings.Join(devices, ","))
		})
	}
}
