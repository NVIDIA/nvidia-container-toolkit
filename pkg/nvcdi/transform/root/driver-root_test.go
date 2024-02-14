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

package root

import (
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"
)

func TestDriverTransformer(t *testing.T) {
	testCases := []struct {
		description      string
		driverRoot       string
		targetDriverRoot string
		devRoot          string
		targetDevRoot    string
		spec             *specs.Spec
		expectedError    error
		expectedSpec     *specs.Spec
	}{
		{
			description:      "dev root not specified",
			driverRoot:       "/driver-root",
			targetDriverRoot: "/host/driver/root/",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/driver-root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/driver-root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/driver/root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/host/driver/root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
		},
		{
			description:      "dev driver root matches",
			driverRoot:       "/driver-root",
			targetDriverRoot: "/host/driver/root/",
			devRoot:          "/driver-root",
			targetDevRoot:    "/host/driver/root/",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/driver-root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/driver-root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/driver/root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/host/driver/root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
		},
		{
			description:      "dev driver root matches separate target dev root",
			driverRoot:       "/driver-root",
			targetDriverRoot: "/host/driver/root/",
			devRoot:          "/driver-root",
			targetDevRoot:    "/host/dev/root/",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/driver-root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/driver-root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/driver/root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/host/dev/root/dev/host/path",
							Path:     "/driver-root/dev/container/path",
						},
					},
				},
			},
		},
		{
			description:      "dev root specified with explicit target",
			driverRoot:       "/driver-root",
			targetDriverRoot: "/host/driver/root/",
			devRoot:          "/",
			targetDevRoot:    "/dev/root/",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/driver-root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/dev/host/path",
							Path:     "/dev/container/path",
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{
							HostPath:      "/host/driver/root/host/path",
							ContainerPath: "/driver-root/container/path",
						},
					},
					DeviceNodes: []*specs.DeviceNode{
						{
							HostPath: "/dev/root/dev/host/path",
							Path:     "/dev/container/path",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			transformer := NewDriverTransformer(
				WithDriverRoot(tc.driverRoot),
				WithTargetDriverRoot(tc.targetDriverRoot),
				WithDevRoot(tc.devRoot),
				WithTargetDevRoot(tc.targetDevRoot),
			)

			err := transformer.Transform(tc.spec)

			require.ErrorIs(t, err, tc.expectedError)
			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}
}
