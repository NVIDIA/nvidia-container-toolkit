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

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestAddNvidiaHook(t *testing.T) {
	logger, logHook := testlog.NewNullLogger()

	mockRuntime := &oci.RuntimeMock{}

	testCases := []struct {
		spec         *specs.Spec
		errorPrefix  string
		shouldNotAdd bool
	}{
		{
			spec: &specs.Spec{},
		},
		{
			spec: &specs.Spec{
				Hooks: &specs.Hooks{},
			},
		},
		{
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{{
						Path: "some-hook",
					}},
				},
			},
		},
		{
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{{
						Path: "nvidia-container-runtime-hook",
					}},
				},
			},
			shouldNotAdd: true,
		},
	}

	for i, tc := range testCases {
		logHook.Reset()

		var numPrestartHooks int
		if tc.spec.Hooks != nil {
			numPrestartHooks = len(tc.spec.Hooks.Prestart)
		}

		shim, err := newNvidiaContainerRuntime(
			logger,
			mockRuntime,
			oci.NewMemorySpec(tc.spec),
		)

		require.NoError(t, err)

		err = shim.Exec([]string{"runtime", "create"})
		require.NoError(t, err)

		if tc.errorPrefix == "" {
			require.NoErrorf(t, err, "%d: %v", i, tc)
		} else {
			require.Truef(t, strings.HasPrefix(err.Error(), tc.errorPrefix), "%d: %v", i, tc)

			require.NotNilf(t, tc.spec.Hooks, "%d: %v", i, tc)
			require.Equalf(t, 1, nvidiaHookCount(tc.spec.Hooks), "%d: %v", i, tc)

			if tc.shouldNotAdd {
				require.Equal(t, numPrestartHooks+1, len(tc.spec.Hooks.Poststart), "%d: %v", i, tc)
			} else {
				require.Equal(t, numPrestartHooks+1, len(tc.spec.Hooks.Poststart), "%d: %v", i, tc)

				nvidiaHook := tc.spec.Hooks.Poststart[len(tc.spec.Hooks.Poststart)-1]

				// TODO: This assumes that the hook has been set up in the makefile
				expectedPath := "/usr/bin/nvidia-container-runtime-hook"
				require.Equalf(t, expectedPath, nvidiaHook.Path, "%d: %v", i, tc)
				require.Equalf(t, []string{expectedPath, "prestart"}, nvidiaHook.Args, "%d: %v", i, tc)
				require.Emptyf(t, nvidiaHook.Env, "%d: %v", i, tc)
				require.Nilf(t, nvidiaHook.Timeout, "%d: %v", i, tc)
			}
		}
	}
}

func TestNvidiaContainerRuntime(t *testing.T) {
	logger, hook := testlog.NewNullLogger()

	mockRuntime := &oci.RuntimeMock{}

	testCases := []struct {
		shouldModify bool
		args         []string
		modifyError  error
		writeError   error
	}{
		{
			shouldModify: false,
		},
		{
			args:         []string{"create"},
			shouldModify: true,
		},
		{
			args:         []string{"--bundle=create"},
			shouldModify: false,
		},
		{
			args:         []string{"--bundle", "create"},
			shouldModify: false,
		},
		{
			args:         []string{"create"},
			shouldModify: true,
		},
		{
			args:         []string{"create"},
			modifyError:  fmt.Errorf("error modifying"),
			shouldModify: true,
		},
		{
			args:         []string{"create"},
			writeError:   fmt.Errorf("error writing"),
			shouldModify: true,
		},
	}

	for i, tc := range testCases {
		hook.Reset()
		specMock := &oci.SpecMock{
			ModifyFunc: func(specModifier oci.SpecModifier) error {
				return tc.modifyError
			},
			FlushFunc: func() error {
				return tc.writeError
			},
		}

		shim, err := newNvidiaContainerRuntime(logger, mockRuntime, specMock)
		require.NoError(t, err)

		err = shim.Exec(tc.args)
		if tc.modifyError != nil || tc.writeError != nil {
			require.Error(t, err, "%d: %v", i, tc)
		} else {
			require.NoError(t, err, "%d: %v", i, tc)
		}

		if tc.shouldModify {
			require.Equal(t, 1, len(specMock.ModifyCalls()), "%d: %v", i, tc)
		} else {
			require.Equal(t, 0, len(specMock.ModifyCalls()), "%d: %v", i, tc)
		}

		writeExpected := tc.shouldModify && tc.modifyError == nil
		if writeExpected {
			require.Equal(t, 1, len(specMock.FlushCalls()), "%d: %v", i, tc)
		} else {
			require.Equal(t, 0, len(specMock.FlushCalls()), "%d: %v", i, tc)
		}
	}
}
