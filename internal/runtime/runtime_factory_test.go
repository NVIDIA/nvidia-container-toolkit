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
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

const (
	runcExecutableName = "runc"
)

func TestMain(m *testing.M) {
	// TEST SETUP
	// Determine the module root and the test binary path
	var err error
	moduleRoot, err := test.GetModuleRoot()
	if err != nil {
		log.Fatalf("error in test setup: could not get module root: %v", err)
	}
	testBinPath := filepath.Join(moduleRoot, "tests", "bin")

	// Set the environment variables for the test
	os.Setenv("PATH", test.PrependToPath(testBinPath, moduleRoot))

	// Confirm that the environment is configured correctly
	runcPath, err := exec.LookPath(runcExecutableName)
	if err != nil || filepath.Join(testBinPath, runcExecutableName) != runcPath {
		log.Fatalf("error in test setup: mock runc path set incorrectly in TestMain(): %v", err)
	}

	// RUN TESTS
	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestFactoryMethod(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	driver := root.New(
		root.WithDriverRoot("/nvidia/driver/root"),
	)

	testCases := []struct {
		description   string
		cfg           *config.Config
		spec          *specs.Spec
		expectedError bool
	}{
		{
			description: "empty config raises error",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{},
			},
			expectedError: true,
		},
		{
			description: "config with runtime raises no error",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
					Mode:     "legacy",
				},
			},
		},
		{
			description: "csv mode is supported",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
					Mode:     "csv",
				},
			},
			spec: &specs.Spec{
				Process: &specs.Process{
					Env: []string{
						"NVIDIA_VISIBLE_DEVICES=all",
					},
				},
			},
		},
		{
			description: "non-legacy discover mode raises error",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
					Mode:     "non-legacy",
				},
			},
			expectedError: true,
		},
		{
			description: "legacy discover mode returns modifier",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
					Mode:     "legacy",
				},
			},
		},
		{
			description: "csv discover mode returns modifier",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
					Mode:     "csv",
				},
			},
		},
		{
			description: "empty mode raises error",
			cfg: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Runtimes: []string{"runc"},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			bundleDir := t.TempDir()

			specFile, err := os.Create(filepath.Join(bundleDir, "config.json"))
			require.NoError(t, err)
			require.NoError(t, json.NewEncoder(specFile).Encode(tc.spec))

			argv := []string{"--bundle", bundleDir, "create"}

			_, err = newNVIDIAContainerRuntime(logger, tc.cfg, argv, driver)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewSpecModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	driver := root.New(
		root.WithDriverRoot("/nvidia/driver/root"),
	)
	testCases := []struct {
		description  string
		config       *config.Config
		spec         *specs.Spec
		expectedSpec *specs.Spec
	}{
		{
			description: "csv mode removes nvidia-container-runtime-hook",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "csv",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: nil,
				},
			},
		},
		{
			description: "csv mode removes nvidia-container-toolkit",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "csv",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: nil,
				},
			},
		},
		{
			description: "cdi mode removes nvidia-container-runtime-hook",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "cdi",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: nil,
				},
			},
		},
		{
			description: "cdi mode removes nvidia-container-toolkit",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "cdi",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: nil,
				},
			},
		},
		{
			description: "legacy mode keeps nvidia-container-runtime-hook",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "legacy",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-runtime-hook",
							Args: []string{"/path/to/nvidia-container-runtime-hook", "prestart"},
						},
					},
				},
			},
		},
		{
			description: "legacy mode keeps nvidia-container-toolkit",
			config: &config.Config{
				NVIDIAContainerRuntimeConfig: config.RuntimeConfig{
					Mode: "legacy",
				},
			},
			spec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
			expectedSpec: &specs.Spec{
				Hooks: &specs.Hooks{
					Prestart: []specs.Hook{
						{
							Path: "/path/to/nvidia-container-toolkit",
							Args: []string{"/path/to/nvidia-container-toolkit", "prestart"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			spec := &oci.SpecMock{
				LoadFunc: func() (*specs.Spec, error) {
					return tc.spec, nil
				},
			}
			m, err := newSpecModifier(logger, tc.config, spec, driver)
			require.NoError(t, err)

			err = m.Modify(tc.spec)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedSpec, tc.spec)
		})
	}
}
