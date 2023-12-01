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

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

func TestDiscoverModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		discover      *discover.DiscoverMock
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description:  "empty discoverer does not modify spec",
			spec:         &specs.Spec{},
			discover:     &discover.DiscoverMock{},
			expectedSpec: &specs.Spec{},
		},
		{
			description: "failed hooks discoverer returns error",
			discover: &discover.DiscoverMock{
				HooksFunc: func() ([]discover.Hook, error) {
					return nil, fmt.Errorf("discover.Hooks error")
				},
			},
			expectedError: fmt.Errorf("discover.Hooks error"),
		},
		{
			description: "discovered hooks are injected into spec",
			spec:        &specs.Spec{},
			discover: &discover.DiscoverMock{
				HooksFunc: func() ([]discover.Hook, error) {
					hooks := []discover.Hook{
						{
							Lifecycle: "prestart",
							Path:      "/hook/a",
							Args:      []string{"/hook/a", "arga"},
						},
						{
							Lifecycle: "createContainer",
							Path:      "/hook/b",
							Args:      []string{"/hook/b", "argb"},
						},
					}
					return hooks, nil
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/hook/a",
							Args: []string{"/hook/a", "arga"},
						},
					},
					CreateContainer: []specs.Hook{
						{
							Path: "/hook/b",
							Args: []string{"/hook/b", "argb"},
						},
					},
				},
			},
		},
		{
			description: "existing hooks are maintained",
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/hook/a",
							Args: []string{"/hook/a", "arga"},
						},
					},
				},
			},
			discover: &discover.DiscoverMock{
				HooksFunc: func() ([]discover.Hook, error) {
					hooks := []discover.Hook{
						{
							Lifecycle: "prestart",
							Path:      "/hook/b",
							Args:      []string{"/hook/b", "argb"},
						},
					}
					return hooks, nil
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/hook/a",
							Args: []string{"/hook/a", "arga"},
						},
						{
							Path: "/hook/b",
							Args: []string{"/hook/b", "argb"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m, err := NewModifierFromDiscoverer(logger, tc.discover)
			require.NoError(t, err)

			err = m.Modify(tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}
}
