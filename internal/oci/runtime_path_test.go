/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/
package oci

import (
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestPathRuntimeConstructor(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	r, err := NewRuntimeForPath(logger, "////an/invalid/path")
	require.Error(t, err)
	require.Nil(t, r)

	r, err = NewRuntimeForPath(logger, "/tmp")
	require.Error(t, err)
	require.Nil(t, r)

	r, err = NewRuntimeForPath(logger, "/dev/null")
	require.Error(t, err)
	require.Nil(t, r)

	r, err = NewRuntimeForPath(logger, "/bin/sh")
	require.NoError(t, err)

	f, ok := r.(*pathRuntime)
	require.True(t, ok)

	require.Equal(t, "/bin/sh", f.path)
}

func TestPathRuntimeForwardsArgs(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		execRuntimeError error
		args             []string
	}{
		{},
		{
			args: []string{"shouldBeReplaced"},
		},
		{
			args: []string{"shouldBeReplaced", "arg1"},
		},
		{
			execRuntimeError: fmt.Errorf("exec error"),
		},
	}

	for _, tc := range testCases {
		mockedRuntime := &RuntimeMock{
			ExecFunc: func(strings []string) error {
				return tc.execRuntimeError
			},
		}
		r := pathRuntime{
			logger:      logger,
			path:        "runtime",
			execRuntime: mockedRuntime,
		}
		err := r.Exec(tc.args)

		require.ErrorIs(t, err, tc.execRuntimeError)

		calls := mockedRuntime.ExecCalls()
		require.Len(t, calls, 1)

		numArgs := len(tc.args)
		if numArgs == 0 {
			numArgs = 1
		}

		require.Len(t, calls[0].Strings, numArgs)
		require.Equal(t, "runtime", calls[0].Strings[0])

		if numArgs > 1 {
			require.EqualValues(t, tc.args[1:], calls[0].Strings[1:])
		}
	}
}
