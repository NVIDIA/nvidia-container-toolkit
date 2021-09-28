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

package runtime

import (
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/modify"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestRuntimeModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		args         []string
		shouldModify bool
	}{
		{},
		{
			args:         []string{"create"},
			shouldModify: true,
		},
	}

	for _, tc := range testCases {
		runtimeMock := &oci.RuntimeMock{}
		specMock := &oci.SpecMock{}
		modifierMock := &modify.ModifierMock{}

		r := NewModifyingRuntimeWrapperWithLogger(
			logger,
			runtimeMock,
			specMock,
			modifierMock,
		)

		err := r.Exec(tc.args)

		require.NoError(t, err)

		expectedCalls := 0
		if tc.shouldModify {
			expectedCalls = 1
		}

		require.Len(t, specMock.LoadCalls(), expectedCalls)
		require.Len(t, modifierMock.ModifyCalls(), expectedCalls)
		require.Len(t, specMock.FlushCalls(), expectedCalls)

		require.Len(t, runtimeMock.ExecCalls(), 1)
	}
}

func TestRuntimeModiferWithLoadError(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runtimeMock := &oci.RuntimeMock{}
	specMock := &oci.SpecMock{
		LoadFunc: specErrorFunc,
	}
	modifierMock := &modify.ModifierMock{}

	r := NewModifyingRuntimeWrapperWithLogger(
		logger,
		runtimeMock,
		specMock,
		modifierMock,
	)

	err := r.Exec([]string{"create"})

	require.Error(t, err)

	require.Len(t, specMock.LoadCalls(), 1)
	require.Len(t, modifierMock.ModifyCalls(), 0)
	require.Len(t, specMock.FlushCalls(), 0)

	require.Len(t, runtimeMock.ExecCalls(), 0)
}

func TestRuntimeModiferWithFlushError(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runtimeMock := &oci.RuntimeMock{}
	specMock := &oci.SpecMock{
		FlushFunc: specErrorFunc,
	}
	modifierMock := &modify.ModifierMock{}

	r := NewModifyingRuntimeWrapperWithLogger(
		logger,
		runtimeMock,
		specMock,
		modifierMock,
	)

	err := r.Exec([]string{"create"})

	require.Error(t, err)

	require.Len(t, specMock.LoadCalls(), 1)
	require.Len(t, modifierMock.ModifyCalls(), 1)
	require.Len(t, specMock.FlushCalls(), 1)

	require.Len(t, runtimeMock.ExecCalls(), 0)
}

func TestRuntimeModiferWithModifyError(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	runtimeMock := &oci.RuntimeMock{}
	specMock := &oci.SpecMock{}
	modifierMock := &modify.ModifierMock{
		ModifyFunc: modifierErrorFunc,
	}

	r := NewModifyingRuntimeWrapperWithLogger(
		logger,
		runtimeMock,
		specMock,
		modifierMock,
	)

	err := r.Exec([]string{"create"})

	require.Error(t, err)

	require.Len(t, specMock.LoadCalls(), 1)
	require.Len(t, modifierMock.ModifyCalls(), 1)
	require.Len(t, specMock.FlushCalls(), 0)

	require.Len(t, runtimeMock.ExecCalls(), 0)

}

func specErrorFunc() error {
	return fmt.Errorf("error")
}

func modifierErrorFunc(oci.Spec) error {
	return fmt.Errorf("error")
}
