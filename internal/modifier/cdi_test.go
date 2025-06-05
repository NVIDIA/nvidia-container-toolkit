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
	"fmt"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

func TestGetAnnotationDevices(t *testing.T) {
	testCases := []struct {
		description     string
		prefixes        []string
		annotations     map[string]string
		expectedDevices []string
		expectedError   error
	}{
		{
			description: "no annotations",
		},
		{
			description: "no matching annotations",
			prefixes:    []string{"not-prefix/"},
			annotations: map[string]string{
				"prefix/foo": "example.com/device=bar",
			},
		},
		{
			description: "single matching annotation",
			prefixes:    []string{"prefix/"},
			annotations: map[string]string{
				"prefix/foo": "example.com/device=bar",
			},
			expectedDevices: []string{"example.com/device=bar"},
		},
		{
			description: "multiple matching annotations",
			prefixes:    []string{"prefix/", "another-prefix/"},
			annotations: map[string]string{
				"prefix/foo":         "example.com/device=bar",
				"another-prefix/bar": "example.com/device=baz",
			},
			expectedDevices: []string{"example.com/device=bar", "example.com/device=baz"},
		},
		{
			description: "multiple matching annotations with duplicate devices",
			prefixes:    []string{"prefix/", "another-prefix/"},
			annotations: map[string]string{
				"prefix/foo":         "example.com/device=bar",
				"another-prefix/bar": "example.com/device=bar",
			},
			expectedDevices: []string{"example.com/device=bar"},
		},
		{
			description: "invalid devices",
			prefixes:    []string{"prefix/"},
			annotations: map[string]string{
				"prefix/foo": "example.com/device",
			},
			expectedError: fmt.Errorf("invalid device %q", "example.com/device"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			devices, err := getAnnotationDevices(tc.prefixes, tc.annotations)
			if tc.expectedError != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.ElementsMatch(t, tc.expectedDevices, devices)
		})
	}
}

func getTestConfig() *config.Config {
	cfg, _ := config.GetDefault()
	return cfg
}

func getTestConfigWithAnnotations() *config.Config {
	cfg, _ := config.GetDefault()
	cfg.NVIDIAContainerRuntimeConfig.Modes.CDI.AnnotationPrefixes = []string{"cdi.k8s.io/"}
	return cfg
}

func TestGetDevicesFromSpec(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description     string
		spec            *specs.Spec
		config          *config.Config
		loadError       error
		expectedDevices []string
		expectedError   string
	}{
		{
			description: "empty spec, no devices specified",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{},
				},
			},
			config:          getTestConfig(),
			expectedDevices: nil,
			expectedError:   "",
		},
		{
			description: "NVIDIA_VISIBLE_DEVICES=all devices specified",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=all",
					},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"nvidia.com/gpu=all"},
			expectedError:   "",
		},
		{
			description: "devices from annotations",
			spec: &specs.Spec{
				Annotations: map[string]string{
					"cdi.k8s.io/test": "example.com/device=device1,example.com/device=device2",
				},
				Process: &specs.Process{
					Env: []string{},
				},
			},
			config:          getTestConfigWithAnnotations(),
			expectedDevices: []string{"example.com/device=device1", "example.com/device=device2"},
		},
		{
			description: "devices from environment variables - single device",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"nvidia.com/gpu=0"},
		},
		{
			description: "devices from environment variables - multiple unique devices",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0,1,2"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"nvidia.com/gpu=0", "nvidia.com/gpu=1", "nvidia.com/gpu=2"},
		},
		{
			description: "devices from environment variables - duplicate devices should be deduplicated",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0,1,0,2,1"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"nvidia.com/gpu=0", "nvidia.com/gpu=1", "nvidia.com/gpu=2"},
		},
		{
			description: "devices from environment variables - qualified device names",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=example.com/device=dev1,0,example.com/device=dev2"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"example.com/device=dev1", "nvidia.com/gpu=0", "example.com/device=dev2"},
		},
		{
			description: "devices from environment variables - duplicate qualified device names should be deduplicated",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=example.com/device=dev1,0,example.com/device=dev1,nvidia.com/gpu=0"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"example.com/device=dev1", "nvidia.com/gpu=0"},
		},
		{
			description: "annotation devices take precedence over environment variables",
			spec: &specs.Spec{
				Annotations: map[string]string{
					"cdi.k8s.io/test": "example.com/device=annotation-device",
				},
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0,1"},
				},
			},
			config:          getTestConfigWithAnnotations(),
			expectedDevices: []string{"example.com/device=annotation-device"},
		},
		{
			description: "devices from environment variables - empty and whitespace devices should be filtered",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=0, ,1, \t ,2"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: []string{"nvidia.com/gpu=0", "nvidia.com/gpu=1", "nvidia.com/gpu=2"},
		},
		{
			description: "devices from environment variables - void should return empty",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=void"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: nil,
		},
		{
			description: "devices from environment variables - none should be filtered out",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=none"},
				},
			},
			config:          getTestConfig(),
			expectedDevices: nil,
		},
		{
			description: "devices from environment variables - all empty devices should result in no devices",
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES= , , \t "},
				},
			},
			config:          getTestConfig(),
			expectedDevices: nil,
		},
		{
			description:   "error loading OCI spec",
			loadError:     fmt.Errorf("failed to load spec"),
			config:        getTestConfig(),
			expectedError: "failed to load OCI spec",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mockSpec := &oci.SpecMock{
				LoadFunc: func() (*specs.Spec, error) {
					if tc.loadError != nil {
						return nil, tc.loadError
					}
					return tc.spec, nil
				},
			}

			devices, err := getDevicesFromSpec(logger, mockSpec, tc.config)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Nil(t, devices)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expectedDevices, devices)
			}

			// Verify that Load was called exactly once
			require.Len(t, mockSpec.LoadCalls(), 1)
		})
	}
}
