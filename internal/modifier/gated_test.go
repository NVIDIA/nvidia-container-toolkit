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

package modifier

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

// fixedDriverVersion allows the driver version to be set without requiring a
// driver installation on the host.
type fixedDriverVersion string

func (v fixedDriverVersion) Version() (string, error) {
	return string(v), nil
}

func TestGetCudaCompatModeModifier(t *testing.T) {
	testLogger, _ := testlog.NewNullLogger()

	const (
		driverVersion     = "580.65.06"
		nvidiaCDIHookPath = "/usr/bin/nvidia-cdi-hook"
	)

	enableCudaCompatHook := specs.Hook{
		Path: nvidiaCDIHookPath,
		Args: []string{"nvidia-cdi-hook", "enable-cuda-compat", "--host-driver-version=" + driverVersion},
		Env:  []string{"NVIDIA_CTK_DEBUG=false"},
	}
	updateLDCacheHook := specs.Hook{
		Path: nvidiaCDIHookPath,
		Args: []string{"nvidia-cdi-hook", "update-ldcache"},
		Env:  []string{"NVIDIA_CTK_DEBUG=false"},
	}

	testCases := map[string]struct {
		mode           string
		cudaCompatHook bool
		expectedHooks  []specs.Hook
	}{
		"legacy mode with the cuda-compat hook updates the ldcache": {
			mode:           "legacy",
			cudaCompatHook: true,
			expectedHooks:  []specs.Hook{enableCudaCompatHook, updateLDCacheHook},
		},
		"legacy mode with the default cuda-compat mode makes no changes": {
			mode: "legacy",
		},
		"cdi mode relies on the discovered ldcache hook": {
			mode:           "cdi",
			cudaCompatHook: true,
			expectedHooks:  []specs.Hook{enableCudaCompatHook},
		},
		"csv mode does not support the cuda-compat hook": {
			mode:           "csv",
			cudaCompatHook: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.GetDefault()
			require.NoError(t, err)
			cfg.NVIDIAContainerRuntimeConfig.Mode = tc.mode
			if tc.cudaCompatHook {
				cfg.NVIDIAContainerRuntimeConfig.Modes.Legacy.CUDACompatMode = config.CUDACompatModeHook
			}

			f := createFactory(
				WithLogger(testLogger),
				WithConfig(cfg),
				WithDriver(root.New(root.WithVersioner(fixedDriverVersion(driverVersion)))),
				WithHookCreator(discover.NewHookCreator(discover.WithNVIDIACDIHookPath(nvidiaCDIHookPath))),
			)

			m, err := f.getCudaCompatModeModifier()
			require.NoError(t, err)
			if tc.expectedHooks == nil {
				require.Nil(t, m)
				return
			}
			require.NotNil(t, m)

			spec := &specs.Spec{}
			require.NoError(t, m.Modify(spec))
			require.NotNil(t, spec.Hooks)
			require.EqualValues(t, tc.expectedHooks, spec.Hooks.CreateContainer)
		})
	}
}
