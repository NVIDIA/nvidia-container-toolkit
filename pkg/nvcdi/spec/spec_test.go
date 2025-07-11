/**
# Copyright 2024 NVIDIA CORPORATION
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

package spec

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform/root"
)

func TestSpec(t *testing.T) {
	minimalSpec := &specs.Spec{
		Kind: "nvidia.com/gpu",
		Devices: []specs.Device{
			{
				Name: "one",
				ContainerEdits: specs.ContainerEdits{
					Env: []string{"DEVICE_FOO=bar"},
				},
			},
		},
	}

	testCases := []struct {
		description      string
		options          []Option
		expectedNewError error
		transform        transform.Transformer
		expectedSpec     string
	}{
		{
			description: "version is overridden",
			options:     []Option{WithVersion("0.8.0"), WithRawSpec(minimalSpec)},
			expectedSpec: `---
cdiVersion: 0.8.0
kind: nvidia.com/gpu
devices:
    - name: one
      containerEdits:
        env:
            - DEVICE_FOO=bar
`,
		},
		{
			description: "raw spec is used as is",
			options: []Option{WithRawSpec(
				&specs.Spec{
					Version: "0.5.0",
					Kind:    "nvidia.com/gpu",
					Devices: []specs.Device{
						{
							Name: "one",
							ContainerEdits: specs.ContainerEdits{
								Env: []string{"DEVICE_FOO=bar"},
							},
						},
					},
				},
			)},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: nvidia.com/gpu
devices:
    - name: one
      containerEdits:
        env:
            - DEVICE_FOO=bar
`,
		},
		{
			description: "raw spec with no version uses minimum version",
			options: []Option{WithRawSpec(
				&specs.Spec{
					Kind: "nvidia.com/gpu",
					Devices: []specs.Device{
						{
							Name: "one",
							ContainerEdits: specs.ContainerEdits{
								Env: []string{"DEVICE_FOO=bar"},
							},
						},
					},
				},
			)},
			expectedSpec: `---
cdiVersion: 0.3.0
kind: nvidia.com/gpu
devices:
    - name: one
      containerEdits:
        env:
            - DEVICE_FOO=bar
`,
		},
		{
			description: "spec with host dev path uses 0.5.0 version",
			options: []Option{WithRawSpec(
				&specs.Spec{
					Kind: "nvidia.com/gpu",
					Devices: []specs.Device{
						{
							Name: "one",
							ContainerEdits: specs.ContainerEdits{
								Env: []string{"DEVICE_FOO=bar"},
							},
						},
					},
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								HostPath: "/some/dev/dev0",
								Path:     "/dev/dev0",
							},
						},
					},
				},
			)},
			expectedSpec: `---
cdiVersion: 0.5.0
kind: nvidia.com/gpu
devices:
    - name: one
      containerEdits:
        env:
            - DEVICE_FOO=bar
containerEdits:
    deviceNodes:
        - path: /dev/dev0
          hostPath: /some/dev/dev0
`,
		},
		{
			description: "transformed spec uses minimum version",
			options: []Option{WithRawSpec(
				&specs.Spec{
					Kind: "nvidia.com/gpu",
					Devices: []specs.Device{
						{
							Name: "one",
							ContainerEdits: specs.ContainerEdits{
								Env: []string{"DEVICE_FOO=bar"},
							},
						},
					},
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								HostPath: "/some/dev/dev0",
								Path:     "/dev/dev0",
							},
						},
					},
				},
			)},
			transform: transform.Merge(
				root.New(
					root.WithRoot("/some/dev/"),
					root.WithTargetRoot("/dev/"),
					root.WithRelativeTo("host"),
				),
				transform.NewSimplifier(),
			),
			expectedSpec: `---
cdiVersion: 0.5.0
kind: nvidia.com/gpu
devices:
    - name: one
      containerEdits:
        env:
            - DEVICE_FOO=bar
containerEdits:
    deviceNodes:
        - path: /dev/dev0
          hostPath: /dev/dev0
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			s, err := New(tc.options...)
			require.ErrorIs(t, err, tc.expectedNewError)

			if tc.transform != nil {
				err := tc.transform.Transform(s.Raw())
				require.NoError(t, err)
			}

			buf := new(bytes.Buffer)

			_, err = s.WriteTo(buf)
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedSpec, buf.String())
		})
	}
}
