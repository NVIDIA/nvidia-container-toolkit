/**
# Copyright 2024 NVIDIA CORPORATION
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

package nvcdi

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestImexMode(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")

	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)
	hostRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-1")

	expectedSpec := `---
cdiVersion: 0.5.0
kind: nvidia.com/imex-channel
devices:
    - name: "0"
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia-caps-imex-channels/channel0
              hostPath: {{ .hostRoot }}/dev/nvidia-caps-imex-channels/channel0
    - name: "1"
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia-caps-imex-channels/channel1
              hostPath: {{ .hostRoot }}/dev/nvidia-caps-imex-channels/channel1
    - name: "2047"
      containerEdits:
        deviceNodes:
            - path: /dev/nvidia-caps-imex-channels/channel2047
              hostPath: {{ .hostRoot }}/dev/nvidia-caps-imex-channels/channel2047
containerEdits:
    env:
        - NVIDIA_VISIBLE_DEVICES=void
`
	expectedSpec = strings.ReplaceAll(expectedSpec, "{{ .hostRoot }}", hostRoot)

	lib, err := New(
		WithLogger(logger),
		WithMode(ModeImex),
		WithDriverRoot(hostRoot),
	)
	require.NoError(t, err)

	spec, err := lib.GetSpec()
	require.NoError(t, err)

	var b bytes.Buffer

	_, err = spec.WriteTo(&b)
	require.NoError(t, err)
	require.Equal(t, expectedSpec, b.String())
}
