/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package discover

import (
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/lookup"
)

func TestCompat32Discoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description    string
		libraries      []string
		locateError    error
		expectedError  bool
		expectedMounts []Mount
		expectedHooks  []Hook
	}{
		{
			description: "libraries are mounted in the compat32 directory",
			libraries: []string{
				"/host/usr/lib/i386-linux-gnu/libcuda.so.999.88.77",
				"/host/usr/lib/i386-linux-gnu/libGLX_nvidia.so.999.88.77",
			},
			expectedMounts: []Mount{
				{
					HostPath: "/host/usr/lib/i386-linux-gnu/libcuda.so.999.88.77",
					Path:     "/usr/lib/nvidia/compat32/libcuda.so.999.88.77",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
				{
					HostPath: "/host/usr/lib/i386-linux-gnu/libGLX_nvidia.so.999.88.77",
					Path:     "/usr/lib/nvidia/compat32/libGLX_nvidia.so.999.88.77",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      testNvidiaCDIHookPath,
					Args:      []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/usr/lib/nvidia/compat32"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
		{
			description: "a host path in a 64-bit directory of the container is not reused",
			libraries: []string{
				"/usr/lib/libcuda.so.999.88.77",
			},
			expectedMounts: []Mount{
				{
					HostPath: "/usr/lib/libcuda.so.999.88.77",
					Path:     "/usr/lib/nvidia/compat32/libcuda.so.999.88.77",
					Options:  []string{"ro", "nosuid", "nodev", "rbind", "rprivate"},
				},
			},
			expectedHooks: []Hook{
				{
					Lifecycle: "createContainer",
					Path:      testNvidiaCDIHookPath,
					Args:      []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/usr/lib/nvidia/compat32"},
					Env:       []string{"NVIDIA_CTK_DEBUG=false"},
				},
			},
		},
		{
			description: "no 32-bit libraries installed",
			locateError: fmt.Errorf("libcuda.so.999.88.77: %w", lookup.ErrNotFound),
		},
		{
			description:   "a lookup error is returned",
			locateError:   fmt.Errorf("lookup failed"),
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			locator := &lookup.LocatorMock{
				LocateFunc: func(pattern string) ([]string, error) {
					require.Equal(t, "*.so.999.88.77", pattern)
					return tc.libraries, tc.locateError
				},
			}
			hookCreator := NewHookCreator(WithNVIDIACDIHookPath(testNvidiaCDIHookPath))
			d := newCompat32Discoverer(logger, locator, "999.88.77", hookCreator)

			mounts, err := d.Mounts()
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMounts, mounts)

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedHooks, hooks)

			devices, err := d.Devices()
			require.NoError(t, err)
			require.Empty(t, devices)
		})
	}
}
