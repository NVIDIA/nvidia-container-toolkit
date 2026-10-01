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

package cdi

import (
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

const (
	managementSpec = `
cdiVersion: 0.5.0
kind: management.nvidia.com/gpu
devices:
- name: all
  containerEdits:
    deviceNodes:
    - path: /dev/nvidia0
      hostPath: /dev/nvidia0
`
	devicePluginSpec = `
cdiVersion: 0.5.0
kind: k8s.device-plugin.nvidia.com/gpu
devices:
- name: GPU-51a7eef3-0eb5-0a96-23bb-c9ae61c95e42
  containerEdits:
    deviceNodes:
    - path: /dev/nvidia0
      hostPath: /dev/nvidia0
`
)

func TestUnresolvedDeviceHint(t *testing.T) {
	testCases := []struct {
		description         string
		specs               map[string]string
		deviceRequestSource string
		unresolved          []string
		expected            string
	}{
		{
			description: "a device under another kind names the kind and the request source",
			specs: map[string]string{
				"management.yaml":    managementSpec,
				"device-plugin.yaml": devicePluginSpec,
			},
			deviceRequestSource: image.DeviceRequestSourceVolumeMounts,
			unresolved:          []string{"management.nvidia.com/gpu=GPU-51a7eef3-0eb5-0a96-23bb-c9ae61c95e42"},
			expected:            "device available as kind k8s.device-plugin.nvidia.com/gpu; requested via volume mounts",
		},
		{
			description: "the request source is reported whatever the mechanism",
			specs: map[string]string{
				"management.yaml":    managementSpec,
				"device-plugin.yaml": devicePluginSpec,
			},
			deviceRequestSource: image.DeviceRequestSourceAnnotations,
			unresolved:          []string{"management.nvidia.com/gpu=GPU-51a7eef3-0eb5-0a96-23bb-c9ae61c95e42"},
			expected:            "device available as kind k8s.device-plugin.nvidia.com/gpu; requested via CDI annotations",
		},
		{
			description: "no kind provides the device, so the available kinds are listed",
			specs: map[string]string{
				"management.yaml": managementSpec,
			},
			unresolved: []string{"management.nvidia.com/gpu=GPU-51a7eef3-0eb5-0a96-23bb-c9ae61c95e42"},
			expected:   "registry contains only kinds management.nvidia.com/gpu",
		},
		{
			description: "an empty registry reports the spec directories searched",
			specs:       map[string]string{},
			unresolved:  []string{"management.nvidia.com/gpu=GPU-51a7eef3-0eb5-0a96-23bb-c9ae61c95e42"},
			expected:    "no CDI specifications found in",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			specDir := t.TempDir()
			for name, contents := range tc.specs {
				require.NoError(t, os.WriteFile(filepath.Join(specDir, name), []byte(contents), 0600))
			}

			registry, err := cdi.NewCache(
				cdi.WithAutoRefresh(false),
				cdi.WithSpecDirs(specDir),
			)
			require.NoError(t, err)

			logger, _ := testlog.NewNullLogger()
			m := fromRegistry{
				logger:              logger,
				registry:            registry,
				deviceRequestSource: tc.deviceRequestSource,
			}

			require.Contains(t, m.unresolvedDeviceHint(tc.unresolved), tc.expected)
		})
	}
}
