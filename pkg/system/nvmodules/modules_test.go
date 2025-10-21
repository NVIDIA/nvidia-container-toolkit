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

package nvmodules

import (
	"errors"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestLoadAll(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runError := errors.New("run error")

	testCases := []struct {
		description   string
		root          string
		runError      error
		expectedError error
		expectedCalls []struct {
			S       string
			Strings []string
		}
	}{
		{
			description: "no root specified",
			root:        "",
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"/sbin/modprobe", []string{"nvidia"}},
				{"/sbin/modprobe", []string{"nvidia-uvm"}},
				{"/sbin/modprobe", []string{"nvidia-modeset"}},
			},
		},
		{
			description: "root causes chroot",
			root:        "/some/root",
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"chroot", []string{"/some/root", "/sbin/modprobe", "nvidia"}},
				{"chroot", []string{"/some/root", "/sbin/modprobe", "nvidia-uvm"}},
				{"chroot", []string{"/some/root", "/sbin/modprobe", "nvidia-modeset"}},
			},
		},
		{
			description:   "run failure is returned",
			root:          "",
			runError:      runError,
			expectedError: runError,
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"/sbin/modprobe", []string{"nvidia"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cmder := &cmderMock{
				RunFunc: func(cmd string, args ...string) error {
					return tc.runError
				},
			}
			m := New(
				WithLogger(logger),
				WithRoot(tc.root),
			)
			m.cmder = cmder

			err := m.LoadAll()
			require.ErrorIs(t, err, tc.expectedError)

			require.EqualValues(t, tc.expectedCalls, cmder.RunCalls())
		})
	}
}

func TestLoad(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runError := errors.New("run error")

	testCases := []struct {
		description   string
		root          string
		module        string
		runError      error
		expectedError error
		expectedCalls []struct {
			S       string
			Strings []string
		}
	}{
		{
			description: "no root specified",
			root:        "",
			module:      "nvidia",
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"/sbin/modprobe", []string{"nvidia"}},
			},
		},
		{
			description: "root causes chroot",
			root:        "/some/root",
			module:      "nvidia",
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"chroot", []string{"/some/root", "/sbin/modprobe", "nvidia"}},
			},
		},
		{
			description:   "run failure is returned",
			root:          "",
			module:        "nvidia",
			runError:      runError,
			expectedError: runError,
			expectedCalls: []struct {
				S       string
				Strings []string
			}{
				{"/sbin/modprobe", []string{"nvidia"}},
			},
		},
		{
			description:   "module prefis is checked",
			module:        "not-nvidia",
			expectedError: errInvalidModule,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cmder := &cmderMock{
				RunFunc: func(cmd string, args ...string) error {
					return tc.runError
				},
			}
			m := New(
				WithLogger(logger),
				WithRoot(tc.root),
			)
			m.cmder = cmder

			err := m.Load(tc.module)
			require.ErrorIs(t, err, tc.expectedError)

			require.EqualValues(t, tc.expectedCalls, cmder.RunCalls())
		})
	}
}
