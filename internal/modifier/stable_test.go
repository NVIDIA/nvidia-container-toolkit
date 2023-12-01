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

package modifier

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

type testConfig struct {
	root    string
	binPath string
}

var cfg *testConfig

func TestMain(m *testing.M) {
	// TEST SETUP
	// Determine the module root and the test binary path
	var err error
	moduleRoot, err := test.GetModuleRoot()
	if err != nil {
		log.Fatalf("error in test setup: could not get module root: %v", err)
	}
	testBinPath := filepath.Join(moduleRoot, "test", "bin")

	// Set the environment variables for the test
	os.Setenv("PATH", test.PrependToPath(testBinPath, moduleRoot))

	// Store the root and binary paths in the test Config
	cfg = &testConfig{
		root:    moduleRoot,
		binPath: testBinPath,
	}

	// RUN TESTS
	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestAddHookModifier(t *testing.T) {
	logger, logHook := testlog.NewNullLogger()

	testHookPath := filepath.Join(cfg.binPath, "nvidia-container-runtime-hook")

	testCases := []struct {
		description   string
		spec          specs.Spec
		expectedError error
		expectedSpec  specs.Spec
	}{
		{
			description: "empty spec adds hook",
			spec:        specs.Spec{},
			expectedSpec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: testHookPath,
							Args: []string{"nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
		},
		{
			description: "spec with empty hooks adds hook",
			spec: specs.Spec{
				Hooks: &specs.Hooks{},
			},
			expectedSpec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: testHookPath,
							Args: []string{"nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
		},
		{
			description: "hook is not replaced",
			spec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "nvidia-container-runtime-hook",
						},
					},
				},
			},
			expectedSpec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "nvidia-container-runtime-hook",
						},
					},
				},
			},
		},
		{
			description: "other hooks are not replaced",
			spec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "some-hook",
						},
					},
				},
			},
			expectedSpec: specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "some-hook",
						},
						{
							Path: testHookPath,
							Args: []string{"nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		logHook.Reset()

		t.Run(tc.description, func(t *testing.T) {

			m := NewStableRuntimeModifier(logger, testHookPath)

			err := m.Modify(&tc.spec)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}

}
