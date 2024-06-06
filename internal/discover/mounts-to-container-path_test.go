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

package discover

import (
	"errors"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

func TestMountsToContainerPath(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	mountOptions := []string{
		"ro",
		"nosuid",
		"nodev",
		"bind",
	}

	testCases := []struct {
		description    string
		required       []string
		locator        lookup.Locator
		containerRoot  string
		expectedMounts []Mount
		expectedError  error
	}{
		{
			description: "containerRoot is prepended",
			required:    []string{"a/path/exists.txt", "another/path/exists.txt"},
			locator: &lookup.LocatorMock{
				LocateFunc: func(s string) ([]string, error) {
					return []string{"/located/root/" + s}, nil
				},
			},
			containerRoot: "/container",
			expectedMounts: []Mount{
				{
					HostPath: "/located/root/a/path/exists.txt",
					Path:     "/container/a/path/exists.txt",
					Options:  mountOptions,
				},
				{
					HostPath: "/located/root/another/path/exists.txt",
					Path:     "/container/another/path/exists.txt",
					Options:  mountOptions,
				},
			},
		},
		{
			description: "duplicate mounts are skipped",
			required:    []string{"a/path/exists.txt", "another/path/exists.txt"},
			locator: &lookup.LocatorMock{
				LocateFunc: func(s string) ([]string, error) {
					return []string{"/located/root/single.txt"}, nil
				},
			},
			containerRoot: "/container",
			expectedMounts: []Mount{
				{
					HostPath: "/located/root/single.txt",
					Path:     "/container/a/path/exists.txt",
					Options:  mountOptions,
				},
			},
		},
		{
			description: "locator errors are ignored",
			required:    []string{"a/path/exists.txt"},
			locator: &lookup.LocatorMock{
				LocateFunc: func(s string) ([]string, error) {
					return nil, errors.New("not found")
				},
			},
			containerRoot:  "/container",
			expectedMounts: []Mount{},
		},
		{
			description: "not located are ignored",
			required:    []string{"a/path/exists.txt"},
			locator: &lookup.LocatorMock{
				LocateFunc: func(s string) ([]string, error) {
					return nil, nil
				},
			},
			containerRoot:  "/container",
			expectedMounts: []Mount{},
		},
		{
			description: "second candidate is ignored",
			required:    []string{"a/path/exists.txt"},
			locator: &lookup.LocatorMock{
				LocateFunc: func(s string) ([]string, error) {
					return []string{"/located/root/" + s, "/located2/root/" + s}, nil
				},
			},
			containerRoot: "/container",
			expectedMounts: []Mount{
				{
					HostPath: "/located/root/a/path/exists.txt",
					Path:     "/container/a/path/exists.txt",
					Options:  mountOptions,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := mountsToContainerPath{
				logger:        logger,
				locator:       tc.locator,
				required:      tc.required,
				containerRoot: tc.containerRoot,
			}

			devices, err := d.Devices()
			require.NoError(t, err)
			require.Empty(t, devices)

			hooks, err := d.Hooks()
			require.NoError(t, err)
			require.Empty(t, hooks)

			mounts, err := d.Mounts()
			require.ErrorIs(t, err, tc.expectedError)
			require.ElementsMatch(t, tc.expectedMounts, mounts)
		})
	}
}
