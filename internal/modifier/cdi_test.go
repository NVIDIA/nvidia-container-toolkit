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
		prefixes        []string
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
			prefixes:    []string{"not-prefix/"},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device=bar",
				},
			},
		},
		{
			description: "single matching annotation",
			prefixes:    []string{"prefix/"},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device=bar",
				},
			},
			expectedDevices: []string{"example.com/device=bar"},
		},
		{
			description: "multiple matching annotations",
			prefixes:    []string{"prefix/", "another-prefix/"},
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
			prefixes:    []string{"prefix/", "another-prefix/"},
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
			prefixes: []string{"prefix/"},
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
			prefixes: []string{"prefix/"},
			spec: &specs.Spec{
				Annotations: map[string]string{
					"prefix/foo": "example.com/device",
				},
			},
			expectedDevices: []string{"nvidia.com/gpu=example.com/device"},
		},
	}

	for _, tc := range testCases {
		tc.input.logger = logger

		image, err := image.NewCUDAImageFromSpec(
			tc.spec,
			image.WithAcceptDeviceListAsVolumeMounts(true),
			image.WithAcceptEnvvarUnprivileged(true),
			image.WithAnnotationsPrefixes(tc.prefixes),
		)
		require.NoError(t, err)
		tc.input.image = image

		t.Run(tc.description, func(t *testing.T) {
			devices := tc.input.DeviceRequests()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedDevices, devices)
		})
	}
}
