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

package root

import (
	"testing"

	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/specs-go"
)

func TestHostRootTransformer(t *testing.T) {
	testCases := []struct {
		description  string
		root         string
		targetRoot   string
		spec         *specs.Spec
		expectedSpec *specs.Spec
	}{
		{
			description:  "nil spec",
			root:         "/root",
			targetRoot:   "/target-root",
			spec:         nil,
			expectedSpec: nil,
		},
		{
			description:  "empty spec",
			root:         "/root",
			targetRoot:   "/target-root",
			spec:         &specs.Spec{},
			expectedSpec: &specs.Spec{},
		},
		{
			description: "device nodes",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{HostPath: "/root/dev/nvidia0", Path: "/root/dev/nvidia0"},
						{HostPath: "/target-root/dev/nvidia1", Path: "/target-root/dev/nvidia1"},
						{HostPath: "/different-root/dev/nvidia2", Path: "/different-root/dev/nvidia2"},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{HostPath: "/target-root/dev/nvidia0", Path: "/root/dev/nvidia0"},
						{HostPath: "/target-root/dev/nvidia1", Path: "/target-root/dev/nvidia1"},
						{HostPath: "/different-root/dev/nvidia2", Path: "/different-root/dev/nvidia2"},
					},
				},
			},
		},
		{
			description: "mounts",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{HostPath: "/root/lib/lib1.so", ContainerPath: "/root/lib/lib1.so"},
						{HostPath: "/target-root/lib/lib2.so", ContainerPath: "/target-root/lib/lib2.so"},
						{HostPath: "/different-root/lib/lib3.so", ContainerPath: "/different-root/lib/lib3.so"},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Mounts: []*specs.Mount{
						{HostPath: "/target-root/lib/lib1.so", ContainerPath: "/root/lib/lib1.so"},
						{HostPath: "/target-root/lib/lib2.so", ContainerPath: "/target-root/lib/lib2.so"},
						{HostPath: "/different-root/lib/lib3.so", ContainerPath: "/different-root/lib/lib3.so"},
					},
				},
			},
		},
		{
			description: "hooks",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							Path: "/root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
						{
							Path: "/target-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/target-root/path/to/target::/target-root/path/to/link",
							},
						},
						{
							Path: "/different-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/different-root/path/to/target::/different-root/path/to/link",
							},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							Path: "/target-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/target-root/path/to/target::/target-root/path/to/link",
							},
						},
						{
							Path: "/target-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/target-root/path/to/target::/target-root/path/to/link",
							},
						},
						{
							Path: "/different-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/different-root/path/to/target::/different-root/path/to/link",
							},
						},
					},
				},
			},
		},
		{
			description: "createContainer hook skips arguments",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createContainer",
							Path:     "/target-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
					},
				},
			},
		},
		{
			description: "startContainer hook skips path and arguments",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "startContainer",
							Path:     "/root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "startContainer",
							Path:     "/root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
					},
				},
			},
		},
		{
			description: "createRuntime hook updates path and arguments",
			root:        "/root",
			targetRoot:  "/target-root",
			spec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createRuntime",
							Path:     "/root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/root/path/to/target::/root/path/to/link",
							},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				ContainerEdits: specs.ContainerEdits{
					Hooks: []*specs.Hook{
						{
							HookName: "createRuntime",
							Path:     "/target-root/usr/bin/nvidia-ctk",
							Args: []string{
								"--link",
								"/target-root/path/to/target::/target-root/path/to/link",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := New(
				WithRoot(tc.root),
				WithTargetRoot(tc.targetRoot),
			).Transform(tc.spec)
			require.NoError(t, err)
			require.Equal(t, tc.spec, tc.expectedSpec)
		})
	}
}
