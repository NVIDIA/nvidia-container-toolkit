/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package modifier

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestDeviceRequests(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description     string
		input           cdiDeviceRequestor
		spec            *specs.Spec
		imageOptions    []image.Option
		expectedDevices []string
	}{
		{
			description: "empty spec yields no devices",
		},
		{
			description: "cdi devices from mounts",
			input: cdiDeviceRequestor{
				defaultKind: "nvidia.com/gpu",
			},
			spec: &specs.Spec{
				Mounts: []specs.Mount{
					{
						Destination: "/var/run/nvidia-container-devices/cdi/nvidia.com/gpu/0",
						Source:      "/dev/null",
					},
					{
						Destination: "/var/run/nvidia-container-devices/cdi/nvidia.com/gpu/1",
						Source:      "/dev/null",
					},
				},
			},
			expectedDevices: []string{"nvidia.com/gpu=0", "nvidia.com/gpu=1"},
		},
		{
			description: "cdi devices from envvar",
			input: cdiDeviceRequestor{
				defaultKind: "nvidia.com/gpu",
			},
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0,example.com/class=device"},
				},
			},
			expectedDevices: []string{"nvidia.com/gpu=0", "example.com/class=device"},
		},
		{
			description: "cdi devices from envvar with default kind",
			input: cdiDeviceRequestor{
				defaultKind: "runtime.nvidia.com/gpu",
			},
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=all"},
				},
			},
			expectedDevices: []string{"runtime.nvidia.com/gpu=all"},
		},
		{
			description: "no matching annotations",
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("not-prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device=bar",
				},
			},
		},
		{
			description: "single matching annotation",
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device=bar",
				},
			},
			expectedDevices: []string{"example.com/device=bar"},
		},
		{
			description: "multiple matching annotations",
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("prefix/", "another-prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo":         "example.com/device=bar",
					"another-prefix/bar": "example.com/device=baz",
				},
			},

			expectedDevices: []string{"example.com/device=baz", "example.com/device=bar"},
		},
		{
			description: "multiple matching annotations with duplicate devices",
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("prefix/", "another-prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo":         "example.com/device=bar",
					"another-prefix/bar": "example.com/device=bar",
				},
			},
			expectedDevices: []string{"example.com/device=bar", "example.com/device=bar"},
		},
		{
			description: "devices in annotations are expanded",
			input: cdiDeviceRequestor{
				defaultKind: "nvidia.com/gpu",
			},
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "device",
				},
			},
			expectedDevices: []string{"nvidia.com/gpu=device"},
		},
		{
			description: "invalid devices in annotations are treated as strings",
			input: cdiDeviceRequestor{
				defaultKind: "nvidia.com/gpu",
			},
			imageOptions: []image.Option{
				image.WithAnnotationsPrefixes("prefix/"),
			},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device",
				},
			},
			expectedDevices: []string{"nvidia.com/gpu=example.com/device"},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES=none",
			input: cdiDeviceRequestor{
				defaultKind: "runtime.nvidia.com/gpu",
			},
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=none"},
				},
			},
			expectedDevices: []string{"runtime.nvidia.com/gpu=none"},
		},
		{
			description: "SWARM_RESOURCE envvar is used over NVIDIA_VISIBLE_DEVICES",
			input: cdiDeviceRequestor{
				defaultKind: "runtime.nvidia.com/gpu",
			},
			imageOptions: []image.Option{
				image.WithPreferredVisibleDevicesEnvVars("SWARM_RESOURCE"),
			},
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=all", "SWARM_RESOURCE=GPU1"},
				},
			},
			expectedDevices: []string{"runtime.nvidia.com/gpu=GPU1"},
		},
	}

	for _, tc := range testCases {
		tc.input.logger = logger

		image, err := image.NewCUDAImageFromSpec(
			tc.spec,
			append(
				[]image.Option{
					// TODO: We should pull these into the testcase options.
					image.WithAcceptDeviceListAsVolumeMounts(true),
					image.WithAcceptEnvvarUnprivileged(true),
				},
				tc.imageOptions...)...,
		)
		require.NoError(t, err)
		tc.input.image = &image

		t.Run(tc.description, func(t *testing.T) {
			devices := tc.input.DeviceRequests()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedDevices, devices)
		})
	}
}

func TestMigCapsDeviceRequests(t *testing.T) {
	testCases := []struct {
		description     string
		env             []string
		expectedDevices []string
	}{
		{
			description: "no MIG envvars yields no devices",
		},
		{
			description:     "monitor devices requested",
			env:             []string{"NVIDIA_MIG_MONITOR_DEVICES=all"},
			expectedDevices: []string{"mode=mig-caps,id=monitor"},
		},
		{
			description:     "config devices requested",
			env:             []string{"NVIDIA_MIG_CONFIG_DEVICES=all"},
			expectedDevices: []string{"mode=mig-caps,id=config"},
		},
		{
			description:     "both config and monitor requested",
			env:             []string{"NVIDIA_MIG_CONFIG_DEVICES=all", "NVIDIA_MIG_MONITOR_DEVICES=all"},
			expectedDevices: []string{"mode=mig-caps,id=config", "mode=mig-caps,id=monitor"},
		},
		{
			description: "empty value yields no devices",
			env:         []string{"NVIDIA_MIG_MONITOR_DEVICES="},
		},
		{
			description: "non-all value is ignored",
			env:         []string{"NVIDIA_MIG_MONITOR_DEVICES=0", "NVIDIA_MIG_CONFIG_DEVICES=0,1"},
		},
		{
			description:     "allowed with whole-GPU visibility",
			env:             []string{"NVIDIA_VISIBLE_DEVICES=0", "NVIDIA_MIG_MONITOR_DEVICES=all"},
			expectedDevices: []string{"mode=mig-caps,id=monitor"},
		},
		{
			description: "ignored when scoped to specific MIG devices (index)",
			env:         []string{"NVIDIA_VISIBLE_DEVICES=0:0", "NVIDIA_MIG_MONITOR_DEVICES=all"},
		},
		{
			description: "ignored when scoped to specific MIG devices (UUID)",
			env:         []string{"NVIDIA_VISIBLE_DEVICES=MIG-GPU-b1028956-cfa2-0990-bf4a-5da9abb51763/3/0", "NVIDIA_MIG_CONFIG_DEVICES=all"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			img, err := image.NewCUDAImageFromSpec(
				&specs.Spec{
					Process: &specs.Process{Env: tc.env},
				},
				image.WithAcceptEnvvarUnprivileged(true),
			)
			require.NoError(t, err)

			devices := migCapsDevices(img).DeviceRequests()
			require.EqualValues(t, tc.expectedDevices, devices)
		})
	}
}

func Test_cdiModeIdentfiersFromDevices(t *testing.T) {
	testCases := []struct {
		description string
		devices     []string
		expected    *cdiModeIdentifiers
	}{
		{
			description: "empty device list",
			devices:     []string{},
			expected: &cdiModeIdentifiers{
				modes:             nil,
				idsByMode:         map[string][]string{},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "single automatic device",
			devices:     []string{"0"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"auto"},
				idsByMode:         map[string][]string{"auto": {"0"}},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "multiple automatic devices",
			devices:     []string{"0", "1"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"auto"},
				idsByMode:         map[string][]string{"auto": {"0", "1"}},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "device with explicit mode",
			devices:     []string{"mode=gds,id=foo"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"gds"},
				idsByMode:         map[string][]string{"gds": {"foo"}},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "mixed auto and explicit",
			devices:     []string{"0", "mode=gds,id=foo", "mode=gdrcopy,id=bar"},
			expected: &cdiModeIdentifiers{
				modes: []string{"auto", "gds", "gdrcopy"},
				idsByMode: map[string][]string{
					"auto":    {"0"},
					"gds":     {"foo"},
					"gdrcopy": {"bar"},
				},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "device with only mode, no id",
			devices:     []string{"mode=nvswitch"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"nvswitch"},
				idsByMode:         map[string][]string{},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "duplicate modes",
			devices:     []string{"mode=gds,id=x", "mode=gds,id=y", "mode=gds"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"gds"},
				idsByMode:         map[string][]string{"gds": {"x", "y"}},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES=none",
			devices:     []string{"none"},
			expected: &cdiModeIdentifiers{
				modes:             []string{"auto"},
				idsByMode:         map[string][]string{"auto": {"none"}},
				deviceClassByMode: map[string]string{"auto": "gpu"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := cdiModeIdentfiersFromDevices(tc.devices...)
			require.EqualValues(t, tc.expected, result)
		})
	}
}
