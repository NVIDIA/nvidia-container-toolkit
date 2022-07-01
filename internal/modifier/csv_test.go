/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestNewCSVModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description    string
		cfg            *config.Config
		spec           oci.Spec
		visibleDevices string
		expectedError  error
		expectedNil    bool
	}{
		{
			description: "spec load error returns error",
			spec: &oci.SpecMock{
				LoadFunc: func() (*specs.Spec, error) {
					return nil, fmt.Errorf("load failed")
				},
			},
			expectedError: fmt.Errorf("load failed"),
		},
		{
			description:    "visible devices not set returns nil",
			visibleDevices: "NOT_SET",
			expectedNil:    true,
		},
		{
			description:    "visible devices empty returns nil",
			visibleDevices: "",
			expectedNil:    true,
		},
		{
			description:    "visible devices 'void' returns nil",
			visibleDevices: "void",
			expectedNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			spec := tc.spec
			if spec == nil {
				spec = &oci.SpecMock{
					LookupEnvFunc: func(s string) (string, bool) {
						if tc.visibleDevices != "NOT_SET" && s == visibleDevicesEnvvar {
							return tc.visibleDevices, true
						}
						return "", false
					},
				}
			}

			m, err := NewCSVModifier(logger, tc.cfg, spec)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectedNil || tc.expectedError != nil {
				require.Nil(t, m)
			} else {
				require.NotNil(t, m)
			}
		})
	}
}

func TestCSVModifierRemovesHook(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description: "modification removes existing nvidia-container-runtime-hook",
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{},
				},
			},
		},
		{
			description: "modification removes existing nvidia-container-toolkit",
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m := nvidiaContainerRuntimeHookRemover{logger: logger}

			err := m.Modify(tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Empty(t, tc.spec.Hooks.Prestart)
		})
	}
}
