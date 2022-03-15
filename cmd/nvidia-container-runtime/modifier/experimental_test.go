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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestConstructor(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		cfg           *config.RuntimeConfig
		expectedError error
	}{
		{
			description:   "empty config raises error",
			cfg:           &config.RuntimeConfig{},
			expectedError: fmt.Errorf("invalid discover mode"),
		},
		{
			description: "non-legacy discover mode raises error",
			cfg: &config.RuntimeConfig{
				DiscoverMode: "non-legacy",
			},
			expectedError: fmt.Errorf("invalid discover mode"),
		},
		{
			description: "legacy discover mode returns modifier",
			cfg: &config.RuntimeConfig{
				DiscoverMode: "legacy",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := NewExperimentalModifier(logger, tc.cfg)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExperimentalModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		discover      *discover.DiscoverMock
		spec          *specs.Spec
		expectedError error
		expectedSpec  *specs.Spec
	}{
		{
			description: "empty discoverer does not modify spec",
			discover:    &discover.DiscoverMock{},
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
		{
			description: "modification fails for existing nvidia-container-runtime-hook",
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
			expectedError: fmt.Errorf("nvidia-container-runtime-hook already exists"),
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
		},
		{
			description: "modification fails for existing nvidia-container-toolkit",
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
			expectedError: fmt.Errorf("nvidia-container-toolkit already exists"),
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m, err := newExperimentalModifierFromDiscoverer(logger, tc.discover)
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
