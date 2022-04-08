/*
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
*/

package runtime

import (
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestExec(t *testing.T) {
	logger, hook := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		shouldLoad    bool
		shouldModify  bool
		shouldFlush   bool
		shouldForward bool
		args          []string
		modifyError   error
		writeError    error
		modifer       oci.SpecModifier
	}{
		{
			description:   "no args forwards",
			shouldLoad:    false,
			shouldModify:  false,
			shouldFlush:   false,
			shouldForward: true,
			modifer:       &modiferMock{},
		},
		{
			description:   "create modifies",
			args:          []string{"create"},
			shouldLoad:    true,
			shouldModify:  true,
			shouldFlush:   true,
			shouldForward: true,
			modifer:       &modiferMock{},
		},
		{
			description:   "modify error does not write or forward",
			args:          []string{"create"},
			modifyError:   fmt.Errorf("error modifying"),
			shouldLoad:    true,
			shouldModify:  true,
			shouldFlush:   false,
			shouldForward: false,
			modifer:       &modiferMock{},
		},
		{
			description:   "write error does not forward",
			args:          []string{"create"},
			writeError:    fmt.Errorf("error writing"),
			shouldLoad:    true,
			shouldModify:  true,
			shouldFlush:   true,
			shouldForward: false,
			modifer:       &modiferMock{},
		},
		{
			description:   "nil modifier forwards on create",
			args:          []string{"create"},
			shouldLoad:    false,
			shouldModify:  false,
			shouldFlush:   false,
			shouldForward: true,
			modifer:       nil,
		},
	}

	for _, tc := range testCases {
		hook.Reset()

		t.Run(tc.description, func(t *testing.T) {
			runtimeMock := &oci.RuntimeMock{}
			specMock := &oci.SpecMock{
				ModifyFunc: func(specModifier oci.SpecModifier) error {
					return tc.modifyError
				},
				FlushFunc: func() error {
					return tc.writeError
				},
			}

			shim := NewModifyingRuntimeWrapper(
				logger,
				runtimeMock,
				specMock,
				// TODO: We should test the interactions with the SpecModifier too
				tc.modifer,
			)

			err := shim.Exec(tc.args)
			if tc.modifyError != nil || tc.writeError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.shouldLoad {
				require.Equal(t, 1, len(specMock.LoadCalls()))
			} else {
				require.Equal(t, 0, len(specMock.LoadCalls()))
			}
			if tc.shouldModify {
				require.Equal(t, 1, len(specMock.ModifyCalls()))
			} else {
				require.Equal(t, 0, len(specMock.ModifyCalls()))
			}
			if tc.shouldFlush {
				require.Equal(t, 1, len(specMock.FlushCalls()))
			} else {
				require.Equal(t, 0, len(specMock.FlushCalls()))
			}
			if tc.shouldForward {
				require.Equal(t, 1, len(runtimeMock.ExecCalls()))
			} else {
				require.Equal(t, 0, len(runtimeMock.ExecCalls()))
			}
		})
	}
}

func TestNilModiferReturnsRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runtimeMock := &oci.RuntimeMock{}
	specMock := &oci.SpecMock{}

	shim := NewModifyingRuntimeWrapper(
		logger,
		runtimeMock,
		specMock,
		nil,
	)

	require.Equal(t, runtimeMock, shim)
}

type modiferMock struct{}

func (m modiferMock) Modify(*specs.Spec) error {
	return nil
}
