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

package nvcdi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"

	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestNvidiaSMISymlinkHook(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	errMounts := errors.New("mounts error")

	testCases := []struct {
		description   string
		mounts        discover.Discover
		expectedError error
		expectedHooks []discover.Hook
	}{
		{
			description: "mounts error is returned",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					return nil, errMounts
				},
			},
			expectedError: errMounts,
		},
		{
			description: "no mounts returns no hooks",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					return nil, nil
				},
			},
		},
		{
			description: "no nvidia-smi returns no hooks",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					mounts := []discover.Mount{
						{Path: "/not-nvidia-smi"},
						{Path: "/also-not-nvidia-smi"},
					}
					return mounts, nil
				},
			},
		},
		{
			description: "nvidia-smi must be in path",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					mounts := []discover.Mount{
						{Path: "/not-nvidia-smi", HostPath: "nvidia-smi"},
						{Path: "/also-not-nvidia-smi", HostPath: "not-nvidia-smi"},
					}
					return mounts, nil
				},
			},
		},
		{
			description: "nvidia-smi returns hook",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					mounts := []discover.Mount{
						{Path: "nvidia-smi"},
					}
					return mounts, nil
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "nvidia-ctk",
					Args: []string{"nvidia-ctk", "hook", "create-symlinks",
						"--link", "nvidia-smi::/usr/bin/nvidia-smi"},
				},
			},
		},
		{
			description: "checks basename",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					mounts := []discover.Mount{
						{Path: "/some/path/nvidia-smi"},
						{Path: "/nvidia-smi/but-not"},
					}
					return mounts, nil
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "nvidia-ctk",
					Args: []string{"nvidia-ctk", "hook", "create-symlinks",
						"--link", "/some/path/nvidia-smi::/usr/bin/nvidia-smi"},
				},
			},
		},
		{
			description: "returns first match",
			mounts: &discover.DiscoverMock{
				MountsFunc: func() ([]discover.Mount, error) {
					mounts := []discover.Mount{
						{Path: "/some/path/nvidia-smi"},
						{Path: "/another/path/nvidia-smi"},
					}
					return mounts, nil
				},
			},
			expectedHooks: []discover.Hook{
				{
					Lifecycle: "createContainer",
					Path:      "nvidia-ctk",
					Args: []string{"nvidia-ctk", "hook", "create-symlinks",
						"--link", "/some/path/nvidia-smi::/usr/bin/nvidia-smi"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m := nvidiaSMISimlinkHook{
				logger:        logger,
				mountsFrom:    tc.mounts,
				nvidiaCTKPath: "nvidia-ctk",
			}

			devices, err := m.Devices()
			require.NoError(t, err)
			require.Empty(t, devices)

			mounts, err := m.Mounts()
			require.NoError(t, err)
			require.Empty(t, mounts)

			hooks, err := m.Hooks()
			require.ErrorIs(t, err, tc.expectedError)
			require.Equal(t, tc.expectedHooks, hooks)
		})
	}
}
