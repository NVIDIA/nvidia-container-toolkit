/**
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
**/

package modifier

import (
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/api/config/v1"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test/to"
)

func TestNewCSVModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	defer devices.SetAllForTest()()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	lookupRoot := filepath.Join(moduleRoot, "testdata", "lookup")

	testCases := []struct {
		description         string
		cfg                 config.Config
		envmap              map[string]string
		driverRootfs        string
		devRootfs           string
		expectedErrorString string
		expectedSpec        specs.Spec
	}{
		{
			description: "visible devices not set returns nil",
			envmap:      map[string]string{},
		},
		{
			description: "visible devices empty returns nil",
			envmap:      map[string]string{"NVIDIA_VISIBLE_DEVICES": ""},
		},
		{
			description: "visible devices 'void' returns nil",
			envmap:      map[string]string{"NVIDIA_VISIBLE_DEVICES": "void"},
		},
		{
			description:  "visible devices all",
			envmap:       map[string]string{"NVIDIA_VISIBLE_DEVICES": "all"},
			driverRootfs: "rootfs-orin",
			expectedSpec: specs.Spec{
				Process: &specs.Process{
					Env: []string{"NVIDIA_VISIBLE_DEVICES=void"},
				},
				Mounts: []specs.Mount{
					{Source: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", Destination: "/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so.1.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
					{Source: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", Destination: "/usr/lib/aarch64-linux-gnu/nvidia/libnvidia-ml.so.1", Options: []string{"ro", "nosuid", "nodev", "rbind", "rprivate"}},
				},
				Hooks: &specs.Hooks{
					CreateContainer: []specs.Hook{
						{
							Path: "/usr/bin/nvidia-cdi-hook",
							Args: []string{"nvidia-cdi-hook", "create-symlinks", "--link", "libcuda.so.1::/usr/lib/aarch64-linux-gnu/nvidia/libcuda.so"},
							Env:  []string{"NVIDIA_CTK_DEBUG=false"},
						},
						{
							Path: "/usr/bin/nvidia-cdi-hook",
							Args: []string{"nvidia-cdi-hook", "update-ldcache", "--folder", "/usr/lib/aarch64-linux-gnu/nvidia"},
							Env:  []string{"NVIDIA_CTK_DEBUG=false"},
						},
					},
				},
				Linux: &specs.Linux{
					Devices: []specs.LinuxDevice{
						{Path: "/dev/nvidia0", Type: "c", Major: 99},
					},
					Resources: &specs.LinuxResources{
						Devices: []specs.LinuxDeviceCgroup{
							{Allow: true, Type: "c", Major: to.Ptr[int64](99), Minor: to.Ptr[int64](0), Access: "rwm"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := image.New(
				image.WithEnvMap(tc.envmap),
			)
			driverRoot := tc.driverRootfs
			if driverRoot != "" {
				driverRoot = filepath.Join(lookupRoot, tc.driverRootfs)
			}
			devRoot := tc.devRootfs
			if devRoot != "" {
				devRoot = filepath.Join(lookupRoot, tc.devRootfs)
			}
			driver := root.New(root.WithDriverRoot(driverRoot), root.WithDevRoot(devRoot))
			// Override the CSV file search path for this root.
			tc.cfg.NVIDIAContainerRuntimeConfig.Modes.CSV.MountSpecPath = filepath.Join(driverRoot, "/etc/nvidia-container-runtime/host-files-for-container.d")

			f := createFactory(
				WithLogger(logger),
				WithDriver(driver),
				WithLogger(logger),
				WithConfig(&tc.cfg),
				WithImage(&image),
			)
			m, err := f.newCSVModifier()
			if tc.expectedErrorString == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErrorString)
				return
			}

			s := specs.Spec{}
			err = list{m}.Modify(&s)
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedSpec, test.StripRoot(s, driverRoot))
		})
	}
}
